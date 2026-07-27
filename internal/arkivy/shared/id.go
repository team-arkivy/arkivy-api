package shared

import (
	"encoding/json"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ID uuid.UUID

func NewID() (ID, error) {
	u, err := uuid.NewV7()
	if err != nil {
		return ID{}, err
	}
	return ID(u), nil
}

func ParseID(s string) (ID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return ID{}, err
	}
	return ID(u), nil
}

func (id ID) String() string { return uuid.UUID(id).String() }

func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(uuid.UUID(id).String())
}

func (id *ID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	*id = ID(u)
	return nil
}

func (id ID) MarshalBSONValue() (byte, []byte, error) {
	typ, data, err := bson.MarshalValue(bson.Binary{
		Subtype: bson.TypeBinaryUUID, // 0x04
		Data:    id[:],
	})
	return byte(typ), data, err
}

func (id *ID) UnmarshalBSONValue(typ byte, data []byte) error {
	var bin bson.Binary
	if err := bson.UnmarshalValue(bson.Type(typ), data, &bin); err != nil {
		return err
	}
	copy((*id)[:], bin.Data)
	return nil
}
