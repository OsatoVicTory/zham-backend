package zham

import (
	"zham-app/models"
)

func Fingerprint(peaks []models.Peak, songID string, targetZoneSize int) (map[uint32][]models.Couple, int) {
	fingerprints := map[uint32][]models.Couple{}
	zonesLength := 0

	for i, anchor := range peaks {
		if i+targetZoneSize >= len(peaks) {
			break
		}

		cnt := 0
		for j := i; j < len(peaks) && j <= i+targetZoneSize; j++ {
			target := peaks[j]

			if uint32((target.Time-anchor.Time)*1000) <= 0xFFF {
				cnt++
			}
		}

		if cnt >= targetZoneSize {
			for j := i; j < len(peaks) && j <= i+targetZoneSize; j++ {
				target := peaks[j]
				zonesLength++

				address := createAddress(anchor, target)
				anchorTimeMs := uint32(anchor.Time * 1000)

				if _, ok := fingerprints[address]; !ok {
					fingerprints[address] = []models.Couple{}
				}

				fingerprints[address] = append(
					fingerprints[address],
					models.Couple{AnchorTimeMs: anchorTimeMs, SongID: songID},
				)
			}
		}
	}

	return fingerprints, zonesLength
}

func createAddress(anchor, target models.Peak) uint32 {
	anchorFreq := int(anchor.Freq)
	targetFreq := int(target.Freq)
	deltaTimeMs := uint32((target.Time - anchor.Time) * 1000)

	address := uint32(anchorFreq<<22) | uint32(targetFreq<<12) | (deltaTimeMs & 0xFFF)

	return address //uint32(address)
}
