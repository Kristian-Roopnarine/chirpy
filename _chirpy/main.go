package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Kristian-Roopnarine/chirpy/internal/database"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type apiConfig struct {
	fileserverHits int
	JwtSecret      string
}

type CustomClaims struct {
	jwt.RegisteredClaims
}

func main() {
	godotenv.Load()
	const port = "8080"
	mux := http.NewServeMux()
	corsMux := corsMiddleware(mux)
	jwtSecret := os.Getenv("JWT_SECRET")
	apiCfg := apiConfig{fileserverHits: 0, JwtSecret: jwtSecret}
	mux.Handle("/app/*", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", healthCheck)
	mux.HandleFunc("/api/reset", apiCfg.reset)
	mux.HandleFunc("GET /admin/metrics", apiCfg.getHits)
	mux.HandleFunc("POST /api/chirps", createChirp)
	mux.HandleFunc("GET /api/chirps", getChirps)
	mux.HandleFunc("GET /api/chirps/{CHIRPID}", getChirp)
	mux.HandleFunc("POST /api/users", createUser)
	mux.HandleFunc("PUT /api/users", updateUser)
	mux.HandleFunc("POST /api/login", login)
	s := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}
	log.Fatal(s.ListenAndServe())
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	errResp := struct {
		Error string `json:"error"`
	}{
		Error: msg,
	}
	dat, err := json.Marshal(errResp)
	if err != nil {
		log.Printf("issue encoding parameters %v", msg)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("issue with encoding payload: %v", payload)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	respBody := body{Email: "", Password: ""}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode((&respBody))
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	db := database.NewDb(database.DbPath)
	_, err = db.Read()
	if err != nil {
		respondWithError(w, 500, "error reading db")
		return
	}
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(respBody.Password), 2)
	if err != nil {
		respondWithError(w, 500, "error with password")
	}
	user, err := db.SaveUser(struct {
		Email    string
		Password string
	}{Email: respBody.Email, Password: string(hashedPass)})
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	respondWithJSON(w, 201, struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
	}{
		Id:    user.Id,
		Email: user.Email,
	})
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	bearerToken := r.Header.Get("Authorization")
	token := strings.Split(bearerToken, " ")[1]
	jwtToken, err := jwt.ParseWithClaims(token, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		jwtSecret := os.Getenv("JWT_SECRET")
		return []byte(jwtSecret), nil
	})
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	claims, ok := jwtToken.Claims.(*CustomClaims)
	if !ok {
		respondWithError(w, 401, "unknown claims type")
		return
	}
	id, _ := strconv.Atoi(claims.Subject)
	fmt.Println(id)
	respBody := body{Email: "", Password: ""}
	decoder := json.NewDecoder(r.Body)

	err = decoder.Decode(&respBody)
	if err != nil {
		respondWithError(w, 500, "error decoding json")
		return
	}
	db := database.NewDb(database.DbPath)
	user, err := db.UpdateUser(id, struct {
		Email    string
		Password string
	}(respBody))
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	if user == (database.User{}) {
		respondWithError(w, 404, "user not found")
		return
	}
	respondWithJSON(w, 200, struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
	}{
		Id:    user.Id,
		Email: user.Email,
	})
}

func login(w http.ResponseWriter, r *http.Request) {
	jwtSecret := os.Getenv("JWT_SECRET")
	type body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		// optional fields
		ExpiresInSeconds int `json:"expires_in_seconds"`
	}
	respBody := body{Email: "", Password: ""}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&respBody)
	if err != nil {
		respondWithError(w, 500, "invalid data")
		return
	}
	db := database.NewDb(database.DbPath)
	user, err := db.GetUserByEmail(respBody.Email)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(respBody.Password))
	if err != nil {
		respondWithError(w, 401, "unauthorized")
		return
	}

	expiresInSeconds := respBody.ExpiresInSeconds
	if expiresInSeconds == 0 {
		expiresInSeconds = 86400
	}
	claims := CustomClaims{
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresInSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "chirpy",
			Subject:   fmt.Sprintf("%d", user.Id),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	respondWithJSON(w, 200, struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
		Token string `json:"token"`
	}{Id: user.Id, Email: user.Email, Token: ss})

}

func createChirp(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Body string `json:"body"`
	}
	respBody := body{Body: ""}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&respBody)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	msg, err := validateChirp(respBody.Body)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	db := database.NewDb(database.DbPath)
	_, err = db.Read()
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	chirp, err := db.SaveChirp(struct{ Body string }{Body: msg})
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	respondWithJSON(w, 201, chirp)
}

func getChirp(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("CHIRPID"))
	if err != nil {
		respondWithError(w, 500, "error parsing request params")
		return
	}
	db := database.NewDb(database.DbPath)
	chirp, err := db.GetChirp(id)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	if chirp == (database.Chirp{}) {
		respondWithJSON(w, 404, "chirp not found")
		return
	}
	dat, err := json.Marshal(chirp)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	w.WriteHeader(200)
	w.Write(dat)

}

func getChirps(w http.ResponseWriter, r *http.Request) {
	db := database.NewDb(database.DbPath)
	chirps, err := db.Read()
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	respondWithJSON(w, 200, chirps)
}

func validateChirp(msg string) (string, error) {
	if len(msg) > 140 || msg == "" {
		return "", errors.New("Chirp is too long")
	}
	cleanedInput := cleanInput(msg)
	return cleanedInput, nil
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Hit")
		cfg.fileserverHits = cfg.fileserverHits + 1
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) getHits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	htmlToRender := `<html>
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
	</html>`
	w.Write([]byte(fmt.Sprintf(htmlToRender, cfg.fileserverHits)))
}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Reset api hit count"))
}
