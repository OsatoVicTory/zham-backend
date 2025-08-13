package db

import (
	"encoding/json"
	"fmt"
	"os"
	"zham-app/models"
)

func findCleanPath() (string, error) {
	filePaths := []string{"db10.json", "db9.json", "db8.json", "db7.json", "db6.json", "db5.json", "db4.json", "db3.json", "db2.json", "db.json"}
	var resPath string

	for _, path := range filePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}

		// var resTmp map[uint32][]models.Couple
		var resTmp map[uint32][]string
		if err := json.Unmarshal(data, &resTmp); err != nil {
			return "", err
		}

		if len(resTmp) > 0 {
			break
		} else {
			resPath = path
		}
	}
	return resPath, nil
}

func ReadFromJSON(filePath string) (map[uint32][]string, error) {
	filePaths := []string{"db.json", "db2.json", "db3.json", "db4.json", "db5.json", "db6.json", "db7.json", "db8.json", "db9.json", "db10.json"}
	// res := make(map[uint32][]models.Couple)
	res := make(map[uint32][]string)

	for _, path := range filePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		// var resTmp map[uint32][]models.Couple
		var resTmp map[uint32][]string
		if err := json.Unmarshal(data, &resTmp); err != nil {
			return nil, err
		}

		for address, couples := range resTmp {
			if _, ok := res[address]; !ok {
				// res[address] = make([]models.Couple, 0)
				res[address] = make([]string, 0)
			}
			res[address] = append(res[address], couples...)
		}
	}

	return res, nil
}

func WriteToJSON(filePath string, dataToStore map[uint32][]models.Couple) error {

	// dbData := map[uint32][]models.Couple{}
	dbData := map[uint32][]string{}

	dbPath, err := findCleanPath()
	if err != nil {
		return err
	}

	for address, couples := range dataToStore {
		if _, ok := dbData[address]; !ok {
			// dbData[address] = make([]models.Couple, 0)
			dbData[address] = make([]string, 0)
		}

		couplesString := make([]string, len(couples))
		for i, c := range couples {
			couplesString[i] = c.SongID + "#" + fmt.Sprint(c.AnchorTimeMs)
		}

		dbData[address] = append(dbData[address], couplesString...)

		// dbData[address] = append(dbData[address], couples...)
	}

	data, err := json.MarshalIndent(dbData, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(dbPath, data, 0644)
}
