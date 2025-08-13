package db

import (
	"strconv"
	"strings"
	"zham-app/models"
)

type Res struct {
	Address uint32
	Couples []models.Couple
}

func GetCouples(addresses []uint32) ([]Res, error) {
	database, err := ReadFromJSON("db.json")
	if err != nil {
		return nil, err
	}

	res := []Res{}
	for _, address := range addresses {

		couples := database[address]
		couplesJson := make([]models.Couple, len(couples))
		for i, c := range couples {
			cSplit := strings.Split(c, "#")
			num, _ := strconv.ParseUint(cSplit[1], 10, 32)
			anchorTimeMs := uint32(num)
			couplesJson[i] = models.Couple{SongID: cSplit[0], AnchorTimeMs: anchorTimeMs}
		}

		// res = append(res, Res{Address: address, Couples: database[address]})
		res = append(res, Res{Address: address, Couples: couplesJson})
	}

	return res, nil
}
