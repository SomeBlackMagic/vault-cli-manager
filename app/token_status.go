package app

import (
	"encoding/json"
	"strings"
	"time"

	"fmt"

	"github.com/cloudfoundry-community/vaultkv"
	"github.com/jhunt/go-ansi"
)

type TokenStatus struct {
	Valid bool
	Info  vaultkv.TokenInfo
}

func (t TokenStatus) String() string {
	if !t.Valid {
		return ansi.Sprintf("@R{Token is invalid}\n")
	}

	retArray := []string{"@G{Token is valid}\n"}
	if t.Info.TTL == time.Duration(0) {
		retArray = append(retArray, "Token has @Y{no expiry}")
	} else {
		retArray = append(retArray, fmt.Sprintf("Token expires in @Y{%s}", time.Until(t.Info.ExpireTime).String()))
	}

	retArray = append(retArray, fmt.Sprintf("Token was created at @Y{%s}", t.Info.CreationTime.Local().Format(time.RFC1123)))

	if t.Info.Renewable {
		retArray = append(retArray, "Token is @G{renewable}")
	} else {
		retArray = append(retArray, "Token is @R{not renewable}")
	}

	if len(t.Info.Policies) == 0 {
		retArray = append(retArray, fmt.Sprintf("Token has @R{no policies}"))
	} else {
		noun := "policy"
		if len(t.Info.Policies) > 1 {
			noun = "policies"
		}
		retArray = append(retArray, fmt.Sprintf("Token has %s @Y{%s}", noun, strings.Join(t.Info.Policies, "}, @Y{")))
	}

	return ansi.Sprintf(strings.Join(retArray, "\n")) + "\n"
}

func (t TokenStatus) MarshalJSON() ([]byte, error) {
	floorZero := func(i int64) int64 {
		if i < 0 {
			i = 0
		}
		return i
	}

	outStruct := struct {
		Valid        bool     `json:"valid"`
		CreationTime int64    `json:"creation_time"`
		ExpireTime   int64    `json:"expire_time"`
		Renewable    bool     `json:"renewable"`
		Policies     []string `json:"policies"`
		TTL          int64    `json:"ttl"`
	}{
		Valid:        t.Valid,
		CreationTime: floorZero(t.Info.CreationTime.Unix()),
		ExpireTime:   floorZero(t.Info.ExpireTime.Unix()),
		Renewable:    t.Info.Renewable,
		Policies:     t.Info.Policies,
		TTL:          floorZero(int64(t.Info.TTL.Seconds())),
	}

	return json.Marshal(&outStruct)
}
