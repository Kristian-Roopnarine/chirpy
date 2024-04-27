package main

import (
	"net/http"
	"sort"
	"strconv"
)

func (cfg *apiConfig) handlerChirpsGet(w http.ResponseWriter, r *http.Request) {
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := strconv.Atoi(chirpIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}
	dbChirp, err := cfg.DB.GetChirp(chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get chirp")
		return
	}

	respondWithJSON(w, http.StatusOK, Chirp{
		ID:   dbChirp.ID,
		Body: dbChirp.Body,
	})
}

func (cfg *apiConfig) handlerChirpsRetrieve(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.DB.GetChirps()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve chirps")
		return
	}
	var authorId int
	_sort := "asc"
	authorIdString := r.URL.Query().Get("author_id")
	sortQuery := r.URL.Query().Get("sort")
	if sortQuery != "" {
		_sort = sortQuery
	}
	filterByAuthorId := authorIdString != ""
	if filterByAuthorId {
		authorId, err = strconv.Atoi(authorIdString)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "error turning author id to int")
			return
		}
	}
	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		if filterByAuthorId && dbChirp.AuthorId != authorId {
			continue
		}
		chirps = append(chirps, Chirp{
			ID:       dbChirp.ID,
			Body:     dbChirp.Body,
			AuthorId: dbChirp.AuthorId,
		})
	}
	sort.Slice(chirps, func(i, j int) bool {
		return sortCondition(_sort, chirps[i].ID, chirps[j].ID)
	})
	respondWithJSON(w, http.StatusOK, chirps)
}

func sortCondition(order string, x, y int) bool {
	if order == "asc" {
		return x < y
	}
	return x > y
}
