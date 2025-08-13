package zham

import (
	"fmt"
	"math"
	"sort"
	"zham-app/db"
	"zham-app/models"
)

type diffStruct struct {
	Diff  int
	Count int
}

// remove in prod
type ColabBody struct {
	Offsets []diffStruct `json:"Offsets"`
	SongID  string       `json:"SongID"`
}

// //

func FindMatches(fingerprints map[uint32][]models.Couple, sizeOfTargetZone int, numTargetZones int) ([]string, []ColabBody, error) {

	addresses := []uint32{}
	for address := range fingerprints {
		addresses = append(addresses, address)
	}

	m, err := db.GetCouples(addresses)
	if err != nil {
		return nil, nil, err
	}

	type matchesStruct struct {
		sampleTimes []models.Couple
		dbTime      uint32
	}

	targetZones := map[string]map[uint32]int{}

	matches := map[string][]matchesStruct{}

	for _, AddressCouples := range m {
		for _, couple := range AddressCouples.Couples {

			if _, ok := targetZones[couple.SongID]; !ok {
				targetZones[couple.SongID] = make(map[uint32]int)
			}
			targetZones[couple.SongID][couple.AnchorTimeMs]++
		}
	}

	for songID, anchorTimes := range targetZones {
		for anchorTime, count := range anchorTimes {
			if count < sizeOfTargetZone {
				delete(targetZones[songID], anchorTime)
			}
		}
	}

	// monitor := map[string]map[uint32]bool{}

	for _, AddressCouples := range m {
		for _, couple := range AddressCouples.Couples {
			cSongID := couple.SongID
			cAnchorTimeMs := couple.AnchorTimeMs

			if _, ok := targetZones[cSongID]; ok {
				if targetZones[cSongID][cAnchorTimeMs] > 0 {
					if _, k := matches[cSongID]; !k {
						matches[cSongID] = make([]matchesStruct, 0)
					}
					matches[cSongID] = append(
						matches[cSongID],
						matchesStruct{sampleTimes: fingerprints[AddressCouples.Address], dbTime: cAnchorTimeMs},
					)
				}
			}
		}
	}

	targetCoefficient := 0.6
	threshold := int(targetCoefficient * float64(numTargetZones))

	type bestStruct struct {
		SongId   string
		maxCount float64
	}
	type padStruct struct {
		SongId   string
		maxCount int
	}

	var bestMatch []bestStruct
	var paddedMatch []padStruct

	offsets := []ColabBody{}

	for songID, anchorZones := range targetZones {
		fmt.Println("songId and anchorZones len", songID, len(anchorZones), threshold)

		if len(anchorZones) >= 1 { //threshold {
			match := matches[songID]

			mp := map[int]int{}

			for _, mtch := range match {
				mpS := map[int]int{}
				for _, sTime := range mtch.sampleTimes {
					diff := int(mtch.dbTime - sTime.AnchorTimeMs)
					mpS[diff]++
				}

				for diff := range mpS {
					mp[diff]++
				}
			}

			arr := make([]diffStruct, len(mp))
			index := 0
			for diff, count := range mp {
				arr[index] = diffStruct{Diff: diff, Count: count}
				index++
			}

			sort.Slice(arr, func(i, j int) bool {
				return arr[i].Diff < arr[j].Diff
			})

			// remove in prod
			var cBody ColabBody
			cBody.Offsets = arr
			cBody.SongID = songID

			offsets = append(offsets, cBody)
			//

			// compute ranges(of length 100) for each diff, for histogram
			mpD := map[int]int{}

			for i := range arr {
				cnt := 0
				for j := i; j < len(arr); j++ {
					if math.Abs(float64(arr[j].Diff-arr[i].Diff)) > 100.0 { // 10
						break
					}
					cnt += arr[j].Count
				}

				mpD[arr[i].Diff] = cnt
			}

			maxCnt, mean, stdDev, z := calculateStats(mpD)
			fmt.Println("song, scores", songID, maxCnt, mean, stdDev, z)
			if z >= 2.5 {
				bestMatch = append(bestMatch, bestStruct{SongId: songID, maxCount: z})
			} else {
				paddedMatch = append(paddedMatch, padStruct{SongId: songID, maxCount: maxCnt})
			}

		}
	}

	sort.Slice(bestMatch, func(i, j int) bool {
		return bestMatch[i].maxCount > bestMatch[j].maxCount
	})

	sort.Slice(paddedMatch, func(i, j int) bool {
		return paddedMatch[i].maxCount > paddedMatch[j].maxCount
	})

	res := make([]string, 10)
	rem := 0

	for _, val := range bestMatch {
		if rem < 10 {
			fmt.Println("bestMatch max count", val.SongId, val.maxCount)
			// res = append(res, val.SongId)
			res[rem] = val.SongId
			rem++
		}
	}

	if rem < 10 {
		for _, score := range paddedMatch {
			if rem < 10 {
				fmt.Println("paddedMatch max count", score.SongId, score.maxCount)
				res[rem] = score.SongId
				rem++
			}
		}
	}

	return res, offsets, nil

}

func calculateStats(data map[int]int) (int, float64, float64, float64) {
	sum := 0
	stdDev := 0.0
	n := len(data)
	maxCnt := 0

	for _, cnt := range data {
		sum += cnt
		maxCnt = max(maxCnt, cnt)
	}
	mean := float64(sum) / float64(n)

	for _, cnt := range data {
		stdDev += math.Pow(float64(cnt)-mean, 2)
	}
	stdDev = math.Sqrt(stdDev / float64(n))

	z := 0.0
	for _, cnt := range data {
		z = max(z, (float64(cnt)-mean)/stdDev)
	}

	return maxCnt, mean, stdDev, z
}
