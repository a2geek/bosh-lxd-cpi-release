package fat32

import (
	"errors"
	"fmt"
	"time"

	"github.com/diskfs/go-diskfs/backend"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/fat12"
)

// MsdosMediaType is the (mostly unused) media type.
type MsdosMediaType uint8

const (
	Media8InchDrDos             MsdosMediaType = 0xe5
	Media525InchTandy           MsdosMediaType = 0xed
	MediaCustomPartitionsDrDos  MsdosMediaType = 0xee
	MediaCustomSuperFloppyDrDos MsdosMediaType = 0xef
	Media35Inch                 MsdosMediaType = 0xf0
	MediaDoubleDensityAltos     MsdosMediaType = 0xf4
	MediaFixedDiskAltos         MsdosMediaType = 0xf5
	MediaFixedDisk              MsdosMediaType = 0xf8
)

// SectorSize is an alias for fat12.SectorSize so that fat32.SectorSize512
// is directly usable in fat12.Dos20BPB struct literals.
type SectorSize = fat12.SectorSize

const (
	SectorSize512        SectorSize = 512
	bytesPerSlot         int        = 32
	maxCharsLongFilename int        = 13
)

//nolint:unused // referenced in future cluster-size validation work
const (
	minClusterSize int = 128
	maxClusterSize int = 65529
)

// FileSystem implements filesystem.FileSystem for FAT32.
// It embeds *fat12.FileSystem to inherit all high-level filesystem methods
// (OpenFile, ReadDir, Mkdir, Remove, Rename, Stat, Label, allocateSpace, …).
// Only FAT32-specific behaviour is implemented here: the dos71EBPB boot
// sector format, the FSInformationSector, and the backup boot sector.
type FileSystem struct {
	*fat12.FileSystem
	fsis     FSInformationSector
	bpbFat32 *dos71EBPB
}

// interface guard
var _ filesystem.FileSystem = (*FileSystem)(nil)

// Type returns filesystem.TypeFat32.
func (fs *FileSystem) Type() filesystem.Type { return filesystem.TypeFat32 }

// SetLabel changes the filesystem volume label. Overrides fat12.FileSystem.SetLabel
// to update the FAT32-format dos71EBPB and write the backup boot sector.
func (fs *FileSystem) SetLabel(volumeLabel string) error {
	if volumeLabel == "" {
		volumeLabel = "NO NAME"
	}
	volumeLabel = fmt.Sprintf("%-11.11s", volumeLabel)

	fs.bpbFat32.volumeLabel = volumeLabel
	if err := fs.WriteBootSector(); err != nil {
		return fmt.Errorf("failed to write the boot sector: %w", err)
	}
	return fs.SetRootDirLabel(volumeLabel)
}

// ── Create ────────────────────────────────────────────────────────────────────

// Create creates a FAT32 filesystem on the given backend.
func Create(b backend.Storage, size, start, blocksize int64, volumeLabel string, reproducible bool) (*FileSystem, error) {
	if blocksize != int64(SectorSize512) && blocksize > 0 {
		return nil, fmt.Errorf("blocksize for FAT32 must be either 512 bytes or 0, not %d", blocksize)
	}
	if size > Fat32MaxSize {
		return nil, fmt.Errorf("requested size is larger than maximum allowed FAT32, requested %d, maximum %d", size, Fat32MaxSize)
	}
	if size < blocksize*4 {
		return nil, fmt.Errorf("requested size is smaller than minimum allowed FAT32, requested %d minimum %d", size, blocksize*4)
	}
	// Check writability early so callers get the plain backend error, not a
	// wrapped one (preserves the error message that existing tests expect).
	if _, err := b.Writable(); err != nil {
		return nil, err
	}

	var volid uint32
	if !reproducible {
		now := time.Now()
		volid = uint32(now.Unix()<<20 | (now.UnixNano() / 1000000))
	}

	fsisPrimarySector := uint16(1)
	backupBootSector := uint16(6)

	var sectorsPerCluster uint8
	switch {
	case size <= 260*MB:
		sectorsPerCluster = 1
	case size <= 8*GB:
		sectorsPerCluster = 8
	case size <= 16*GB:
		sectorsPerCluster = 32
	case size <= 32*GB:
		sectorsPerCluster = 64
	case size <= Fat32MaxSize:
		sectorsPerCluster = 128
	}

	totalSectors := uint32(size / int64(SectorSize512))
	reservedSectors := uint16(32)
	dataSectors := totalSectors - uint32(reservedSectors)
	totalClusters := dataSectors / uint32(sectorsPerCluster)
	sectorsPerFat := uint16(totalClusters / 128)
	mediaType := uint8(MediaFixedDisk)

	fatIDbase := uint32(0x0f << 24)
	fatID := fatIDbase + 0xffff00 + uint32(mediaType)

	dos20bpb := fat12.Dos20BPB{
		SectorsPerCluster:    sectorsPerCluster,
		ReservedSectors:      reservedSectors,
		FatCount:             2,
		TotalSectors:         0,
		MediaType:            mediaType,
		BytesPerSector:       SectorSize512,
		RootDirectoryEntries: 0,
		SectorsPerFat:        0,
	}
	dos331bpb := fat12.Dos331BPB{
		Dos20BPB:        &dos20bpb,
		TotalSectors32:  totalSectors,
		Heads:           1,
		SectorsPerTrack: 1,
		HiddenSectors:   0,
	}
	ebpb := &dos71EBPB{
		Dos331BPB:             &dos331bpb,
		version:               fatVersion0,
		rootDirectoryCluster:  2,
		fsInformationSector:   fsisPrimarySector,
		backupBootSector:      backupBootSector,
		bootFileName:          [12]byte{},
		extendedBootSignature: longDos71EBPB,
		volumeSerialNumber:    volid,
		volumeLabel:           "NO NAME    ",
		fileSystemType:        fileSystemTypeFAT32,
		mirrorFlags:           0,
		reservedFlags:         0,
		driveNumber:           128,
		sectorsPerFat:         uint32(sectorsPerFat),
	}

	fsis := FSInformationSector{
		lastAllocatedCluster:  0xffffffff,
		freeDataClustersCount: 0xffffffff,
	}

	eocMarker := uint32(0x0fffffff)
	fatPrimaryStart := uint64(reservedSectors) * uint64(SectorSize512)
	fatSize := uint32(sectorsPerFat) * uint32(SectorSize512)
	fatSecondaryStart := fatPrimaryStart + uint64(fatSize)
	maxCluster := fatSize / 4
	rootDirCluster := uint32(2)
	clusters := make([]uint32, maxCluster+1)
	clusters[rootDirCluster] = eocMarker
	fat := &table{
		fatID:          fatID,
		eocMarker:      eocMarker,
		unusedMarker:   0,
		size:           fatSize,
		rootDirCluster: rootDirCluster,
		clusters:       clusters,
		maxCluster:     maxCluster,
	}

	dataStart := uint32(fatSecondaryStart) + fatSize
	bytesPerCluster := int(sectorsPerCluster) * int(SectorSize512)

	// Build the base fat12.FileSystem (nil bpb — FAT32 manages its own boot sector).
	base := fat12.NewFileSystem(b, nil, fat,
		dataStart, bytesPerCluster, size, start,
		0, 0, // no fixed root directory (FAT32 uses cluster chain)
		fatPrimaryStart, fatSecondaryStart)

	fs := &FileSystem{
		FileSystem: base,
		fsis:       fsis,
		bpbFat32:   ebpb,
	}
	// Wire up the hooks so inherited methods use FAT32's implementations.
	base.WriteBootSectorFn = fs.writeBootSector
	base.AfterWriteFAT = fs.writeFsis

	// Write boot sector.
	if err := fs.WriteBootSector(); err != nil {
		return nil, fmt.Errorf("failed to write the boot sector: %w", err)
	}

	// Write FAT (also triggers writeFsis via AfterWriteFAT hook).
	if err := fs.WriteFat(); err != nil {
		return nil, fmt.Errorf("failed to write FAT: %w", err)
	}

	// Zero out the root directory cluster.
	writableFile, err := b.Writable()
	if err != nil {
		return nil, err
	}
	clusterStart := fs.Start() + int64(fs.DataStart())
	tmpb := make([]byte, fs.BytesPerCluster())
	written, err := writableFile.WriteAt(tmpb, clusterStart)
	if err != nil {
		return nil, fmt.Errorf("failed to zero out root directory: %w", err)
	}
	if written != fs.BytesPerCluster() {
		return nil, fmt.Errorf("incomplete zero out of root directory, wrote %d bytes instead of %d",
			written, fs.BytesPerCluster())
	}

	if err := fs.SetLabel(volumeLabel); err != nil {
		return nil, fmt.Errorf("failed to set volume label to '%s': %w", volumeLabel, err)
	}

	return fs, nil
}

// ── Read ──────────────────────────────────────────────────────────────────────

// Read reads a FAT32 filesystem from the backend.
func Read(b backend.Storage, size, start, blocksize int64) (*FileSystem, error) {
	if blocksize != 0 && blocksize != int64(SectorSize512) && blocksize != 4096 {
		return nil, fmt.Errorf("blocksize for FAT32 must be 0, 512, or 4096 bytes, not %d", blocksize)
	}
	if size > Fat32MaxSize {
		return nil, fmt.Errorf("requested size is larger than maximum allowed FAT32 size %d", Fat32MaxSize)
	}
	if size < blocksize*4 {
		return nil, fmt.Errorf("requested size is smaller than minimum allowed FAT32 size %d", blocksize*4)
	}

	maxSectorSize := 4096
	bsb := make([]byte, maxSectorSize)
	n, err := b.ReadAt(bsb, start)
	if err != nil {
		return nil, fmt.Errorf("could not read bytes from file: %w", err)
	}
	if n < int(SectorSize512) {
		return nil, fmt.Errorf("only could read %d bytes from file, need at least %d", n, SectorSize512)
	}

	bs, err := msDosBootSectorFromBytes(bsb[:SectorSize512])
	if err != nil {
		return nil, fmt.Errorf("error reading MS-DOS Boot Sector: %w", err)
	}

	bytesPerSector := bs.biosParameterBlock.Dos331BPB.Dos20BPB.BytesPerSector
	sectorsPerFat := bs.biosParameterBlock.sectorsPerFat
	fatSize := sectorsPerFat * uint32(bytesPerSector)
	reservedSectors := bs.biosParameterBlock.Dos331BPB.Dos20BPB.ReservedSectors
	sectorsPerCluster := bs.biosParameterBlock.Dos331BPB.Dos20BPB.SectorsPerCluster
	fatPrimaryStart := uint64(reservedSectors) * uint64(bytesPerSector)
	fatSecondaryStart := fatPrimaryStart + uint64(fatSize)

	fsisBytes := make([]byte, 512)
	if _, err := b.ReadAt(fsisBytes,
		int64(bs.biosParameterBlock.fsInformationSector)*int64(bytesPerSector)+start); err != nil {
		return nil, fmt.Errorf("unable to read bytes for FSInformationSector: %w", err)
	}
	fsis, err := fsInformationSectorFromBytes(fsisBytes)
	if err != nil {
		return nil, fmt.Errorf("error reading FileSystem Information Sector: %w", err)
	}

	partitionTableBytes := make([]byte, fatSize)
	_, _ = b.ReadAt(partitionTableBytes, int64(fatPrimaryStart)+start)
	fat := tableFromBytes(partitionTableBytes)

	_, _ = b.ReadAt(partitionTableBytes, int64(fatSecondaryStart)+start)
	fat2 := tableFromBytes(partitionTableBytes)
	if !fat.equal(fat2) {
		return nil, errors.New("fat tables did not match")
	}
	dataStart := uint32(fatSecondaryStart) + fat.size
	bytesPerCluster := int(sectorsPerCluster) * int(bytesPerSector)

	base := fat12.NewFileSystem(b, nil, fat,
		dataStart, bytesPerCluster, size, start,
		0, 0,
		fatPrimaryStart, fatSecondaryStart)

	fs := &FileSystem{
		FileSystem: base,
		fsis:       *fsis,
		bpbFat32:   bs.biosParameterBlock,
	}
	base.WriteBootSectorFn = fs.writeBootSector
	base.AfterWriteFAT = fs.writeFsis

	return fs, nil
}

// ── FAT32-specific internal helpers ──────────────────────────────────────────

// writeBootSector writes the FAT32-format boot sector (primary + backup).
// It is set as fat12.FileSystem.WriteBootSectorFn so that inherited methods
// such as SetLabel use it automatically.
func (fs *FileSystem) writeBootSector() error {
	writableFile, err := fs.Backend().Writable()
	if err != nil {
		return err
	}
	bs := msDosBootSector{
		oemName:            "godiskfs",
		jumpInstruction:    [3]byte{0xeb, 0x58, 0x90},
		bootCode:           []byte{},
		biosParameterBlock: fs.bpbFat32,
	}
	b, err := bs.toBytes()
	if err != nil {
		return fmt.Errorf("error converting MS-DOS Boot Sector to bytes: %w", err)
	}
	count, err := writableFile.WriteAt(b, fs.Start())
	if err != nil {
		return fmt.Errorf("error writing MS-DOS Boot Sector to disk: %w", err)
	}
	if count != int(SectorSize512) {
		return fmt.Errorf("wrote %d bytes of MS-DOS Boot Sector instead of expected %d", count, SectorSize512)
	}
	if fs.bpbFat32.backupBootSector > 0 {
		bps := int64(fs.bpbFat32.Dos331BPB.Dos20BPB.BytesPerSector)
		count, err = writableFile.WriteAt(b,
			int64(fs.bpbFat32.backupBootSector)*bps+fs.Start())
		if err != nil {
			return fmt.Errorf("error writing backup MS-DOS Boot Sector to disk: %w", err)
		}
		if count != int(SectorSize512) {
			return fmt.Errorf("wrote %d bytes of backup MS-DOS Boot Sector instead of expected %d", count, SectorSize512)
		}
	}
	return nil
}

// writeFsis writes the FSInformationSector to disk (primary + backup copy).
// It is set as fat12.FileSystem.AfterWriteFAT so it is flushed after every
// FAT change.
func (fs *FileSystem) writeFsis() error {
	bpb := fs.bpbFat32
	bps := int64(bpb.Dos331BPB.Dos20BPB.BytesPerSector)
	fsisPrimary := int64(bpb.fsInformationSector) * bps

	fsisBytes := fs.fsis.toBytes()
	writableFile, err := fs.Backend().Writable()
	if err != nil {
		return err
	}
	if _, err := writableFile.WriteAt(fsisBytes, fsisPrimary+fs.Start()); err != nil {
		return fmt.Errorf("unable to write primary FSIS: %w", err)
	}
	if bpb.backupBootSector > 0 {
		if _, err := writableFile.WriteAt(fsisBytes,
			int64(bpb.backupBootSector+1)*bps+fs.Start()); err != nil {
			return fmt.Errorf("unable to write backup FSIS: %w", err)
		}
	}
	return nil
}
