package main

import (
	"encoding/json"
	"time"

	"github.com/cppforlife/bosh-cpi-go/apiv1"
)

type ActualVMMeta struct {
	InstanceGroup string    `json:"instance_group"`
	Job           string    `json:"job"`
	Director      string    `json:"director"`
	Deployment    string    `json:"deployment"`
	Index         string    `json:"index"` // technically this is a string
	CreatedAt     time.Time `json:"created_at"`
}

type ActualDiskMeta struct {
	Director      string    `json:"director"`
	Deployment    string    `json:"deployment"`
	InstanceID    string    `json:"instance_id"`
	InstanceIndex string    `json:"instance_index"` // technically this is a string
	InstanceGroup string    `json:"instance_group"`
	AttachedAt    time.Time `json:"attached_at"`
}

func NewActualVMMeta(metadata apiv1.VMMeta) (ActualVMMeta, error) {
	bytes, err := metadata.MarshalJSON()
	if err != nil {
		return ActualVMMeta{}, err
	}

	actual := ActualVMMeta{}
	if err = json.Unmarshal(bytes, &actual); err != nil {
		return ActualVMMeta{}, err
	}

	return actual, nil
}

func NewActualDiskMeta(metadata apiv1.DiskMeta) (ActualDiskMeta, error) {
	bytes, err := metadata.MarshalJSON()
	if err != nil {
		return ActualDiskMeta{}, err
	}

	actual := ActualDiskMeta{}
	if err = json.Unmarshal(bytes, &actual); err != nil {
		return ActualDiskMeta{}, err
	}

	return actual, nil
}
