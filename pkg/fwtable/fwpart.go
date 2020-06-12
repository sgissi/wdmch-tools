package fwtable

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

var partTypes = []string{
	"reserved",
	"firmware",
	"filesystem",
}

var partFileSystemTypes = []string{
	"jffs2",
	"yaffs2",
	"squash",
	"rawfile",
	"ext4",
	"ubifs",
	"none",
	"unknown",
}

// Size of the file structure in bytes
const partEntrySize = 48

// PartEntry represents one partition in the table.
type PartEntry struct {
	Type, FwType string
	ReadOnly     bool
	Lenght       uint64
	FwCount      int
	EmmcPartID   int
	MountPoint   string
	content      []byte
	table        struct {
		Type       byte
		Options    byte
		Lenght     uint64
		FwCount    uint8
		FwType     uint8
		EmmcPartID uint8
		Reserved   [3]byte
		MountPoint [32]byte
	}
}

func (part *PartEntry) WriteContent() error {
	match := false
	for i, v := range partTypes {
		if part.Type == v {
			part.table.Type = uint8(i)
			match = true
			break
		}
	}
	if !match {
		return fmt.Errorf("invalid type '%s'", part.Type)
	}

	if part.ReadOnly {
		part.table.Options = (1 << 7)
	} else {
		part.table.Options = 0
	}

	part.table.Lenght = part.Lenght
	part.table.FwCount = uint8(part.FwCount)

	match = false
	for i, v := range partFileSystemTypes {
		if part.FwType == v {
			part.table.FwType = uint8(i)
			match = true
			break
		}
	}
	if !match {
		return fmt.Errorf("invalid filesystem type '%s'", part.Type)
	}

	part.table.EmmcPartID = uint8(part.EmmcPartID)
	// Reserved
	for i := 0; i < 3; i++ {
		part.table.Reserved[i] = 0
	}

	mp := []byte(part.MountPoint)
	if len(mp) > 32 {
		return fmt.Errorf("mount point '%s' too long (%d, max is 31)", part.MountPoint, len(mp))
	}
	for i := 0; i < len(mp); i++ {

		part.table.MountPoint[i] = mp[i]
	}
	// Pad with zeros
	for i := len(mp); i < 32; i++ {
		part.table.MountPoint[i] = 0
	}
	buffer := new(bytes.Buffer)
	err := binary.Write(buffer, binary.LittleEndian, &part.table)
	if err != nil {
		return fmt.Errorf("writing: %v", err)
	}
	copy(part.content, buffer.Bytes())

	return nil
}

func (part *PartEntry) ReadContent() error {
	err := binary.Read(bytes.NewReader(part.content), binary.LittleEndian, &part.table)
	if err != nil {
		return err
	}
	if int(part.table.Type) >= len(partTypes) {
		part.Type = fmt.Sprintf("Invalid (%d)", part.table.Type)
	} else {
		part.Type = partTypes[part.table.Type]
	}

	if int(part.table.FwType) >= len(partFileSystemTypes) {
		part.FwType = fmt.Sprintf("Invalid (%d)", part.table.FwType)
	} else {
		part.FwType = partFileSystemTypes[part.table.FwType]
	}

	part.ReadOnly = (part.table.Options & (1 << 7)) == (1 << 7)
	part.Lenght = part.table.Lenght
	part.FwCount = int(part.table.FwCount)
	part.EmmcPartID = int(part.EmmcPartID)
	part.MountPoint = string(part.table.MountPoint[:])
	return nil
}
