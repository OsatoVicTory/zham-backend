package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"zham-app/db"
	"zham-app/wav"
	"zham-app/zham"

	"github.com/gorilla/mux"
)

func main() {
	fmt.Println("Zham!")

	router := mux.NewRouter()

	router.HandleFunc("/zham", searchForSongMatch()).Methods("POST")
	router.HandleFunc("/zham", insertSong()).Methods("PUT")
	router.HandleFunc("/zham/{songId}", getSongZhams()).Methods("GET")

	enhancedRouter := enableCORS(jsonContentTypeMiddleware(router))

	log.Fatal(http.ListenAndServe(":3030", enhancedRouter))
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

type JsonBody struct {
	AudioSample []float64 `json:"audioSample"`
	SampleRate  int       `json:"sampleRate"`
	SongId      string    `json:"SongId"`
	// audioDuration float64
}

func getSongZhams() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		songId := vars["songId"]

		res := 0
		cnt, err := db.ReadNumZham("zham.json", songId)
		if err != nil {
			log.Fatal(err)
		} else {
			res = cnt
		}

		json.NewEncoder(w).Encode(res)

	}
}

func insertSong() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// startTime := time.Now()

		resampleRate := 48000

		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			log.Fatal(err)
		}

		songId := r.FormValue("SongId")
		res, err := wav.ConverterToWAV(r, resampleRate)
		if err != nil {
			log.Fatal(err)
		}

		// json.NewEncoder(w).Encode(res)

		spectrogram, timeArr, err := zham.Spectrogram(res, resampleRate)
		if err != nil {
			log.Fatal(err)
		}

		// peaks := zham.ExtractPeaks(spectrogram, timeArr, 1.0)
		peaks := zham.GetPeaks(spectrogram, timeArr, 11, 5, 1.0)

		fingerprints, _ := zham.Fingerprint(peaks, songId, 5)
		// fmt.Println("fp_len", len(fingerprints))
		if err := db.WriteToJSON("db.json", fingerprints); err != nil {
			log.Fatal(err)
		}

		// fmt.Println("time taken to save song: ", time.Since(startTime))

		json.NewEncoder(w).Encode("Success!")
	}
}

func searchForSongMatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// startTime := time.Now()

		resampleRate := 48000

		// r.Body = http.MaxBytesReader(w, r.Body, int64(50*1024*1024))

		// var body JsonBody
		// decoder := json.NewDecoder(r.Body)
		// err := decoder.Decode(&body)
		// if err != nil {
		// 	log.Fatal(err)
		// 	// http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		// }

		// spectrogram, timeArr, err := zham.Spectrogram(body.AudioSample, body.SampleRate)
		// if err != nil {
		// 	log.Fatal(err)
		// 	// fmt.Errorf("failed to extract spectrogram, %s", err)
		// 	// return
		// }

		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			log.Fatal(err)
		}

		songId := r.FormValue("SongId")
		samples, err := wav.ConverterToWAV(r, resampleRate)
		if err != nil {
			log.Fatal(err)
		}

		spectrogram, timeArr, err := zham.Spectrogram(samples, resampleRate)
		if err != nil {
			log.Fatal(err)
			// fmt.Errorf("failed to extract spectrogram, %s", err)
			// return
		}

		peaks := zham.GetPeaks(spectrogram, timeArr, 11, 5, 1.0)

		fingerprints, numTargetZones := zham.Fingerprint(peaks, songId, 5)

		// res, offsets, err := zham.FindMatches(fingerprints, 5, numTargetZones)
		res, err := zham.FindMatches(fingerprints, 5, numTargetZones)
		if err != nil {
			log.Fatal(err)
		}

		// fmt.Println("full time taken to search song: ", time.Since(startTime))

		// type ResBody struct {
		// 	Results []string
		// 	Peaks   []models.Peak
		// 	Offsets []zham.ColabBody
		// }

		// Res := ResBody{Results: res, Peaks: peaks, Offsets: offsets}

		cnt, err := db.WriteToZhamJSON("zham.json", res[0])
		if err != nil {
			log.Fatal(err)
		}

		type ResBody struct {
			Results   []string
			ZhamCount int
		}

		Res := ResBody{Results: res, ZhamCount: cnt}

		json.NewEncoder(w).Encode(Res)
	}
}
