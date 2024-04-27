package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Kristian-Roopnarine/chirpy/internal/auth"
	"github.com/Kristian-Roopnarine/chirpy/internal/database"
)

func (cfg *apiConfig) handlerChirpDelete(w http.ResponseWriter, r *http.Request) {

	// get userID from jwt token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT")
		return
	}
	userIDString, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT")
		return
	}
	userID, err := strconv.Atoi(userIDString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't decode user id")
		return
	}
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := strconv.Atoi(chirpIDString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Issue with chirp id")
		return
	}

	chirp, err := cfg.DB.GetChirp(chirpID)
	if err != nil {
		if errors.Is(err, database.ErrNotExist) {
			respondWithError(w, http.StatusNotFound, "Chirp not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "issue finding chirp")
		return
	}
	if chirp.AuthorId != userID {
		respondWithError(w, http.StatusForbidden, "forbidden")
		return
	}

	err = cfg.DB.DeleteChirp(chirp.ID, userID)
	if err != nil {
		if errors.Is(err, database.ErrAccessDenied) {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		if errors.Is(err, database.ErrNotExist) {
			respondWithError(w, http.StatusNotFound, "chirp not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "couldn't delete chirp")
		return
	}
	respondWithJSON(w, http.StatusOK, "chirp delete")

}
