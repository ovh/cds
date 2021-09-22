package redis

import (
	"encoding/json"
	"fmt"

	"github.com/ovh/cds/sdk"
)

type Line struct {
	Number float64 `json:"number"`
	Value  string  `json:"value"`
}

func (l Line) Format(f sdk.CDNReaderFormat) ([]byte, error) {
	switch f {
	case sdk.CDNReaderFormatJSON:
		bs, err := json.Marshal(l)
		return bs, sdk.WithStack(err)
	case sdk.CDNReaderFormatText:
		return []byte(l.Value), nil
	}
	return nil, sdk.WithStack(fmt.Errorf("invalid given reader format '%s'", f))
}
