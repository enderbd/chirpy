package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/enderbd/chirpy/internal/auth"
	"github.com/enderbd/chirpy/internal/database"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	IsChirpyRed bool `json:"is_chirpy_red"`
}


func (cfg* apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var params parameters
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not decode parameters", err)
		return
	}

	hashedPasswd, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not hash the password", err)
		return
	}
	dbUser, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email: params.Email,
		HashedPassword: hashedPasswd,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create dbUser", err)
		return
	}

	outUser := User {
		ID: dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email: dbUser.Email,
	}

	respondWithJson(w, http.StatusCreated, outUser)

}


func (cfg* apiConfig) handlerLoginUser(w http.ResponseWriter, r *http.Request) {
	type userRequest struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	var userReq userRequest
	err := json.NewDecoder(r.Body).Decode(&userReq)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not decode parameters", err)
		return
	}

	dbUser, err := cfg.db.GetUserByEmail(r.Context(), userReq.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	}

	match, err := auth.CheckPasswordHash(userReq.Password, dbUser.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not match the password with the hash", err)
		return
	}

	if !match {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	}

	expirationTime := time.Hour

	token, err := auth.MakeJWT(dbUser.ID, cfg.secret, expirationTime)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create JWT token", err)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create a refresh token", err)
		return
	}

	_ , err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token: refreshToken,
		UserID: dbUser.ID,
		ExpiresAt: time.Now().UTC().Add(60 * 24 * time.Hour),

	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not add the refresh token to the table", err)
		return
	}

	outUser := User {
		ID: dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email: dbUser.Email,
		Token: token,
		RefreshToken: refreshToken,
		IsChirpyRed: dbUser.IsChirpyRed,
	}

	respondWithJson(w, http.StatusOK, outUser)


}



func (cfg* apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not find a refresh token in the header", err)
		return
	}

	dbUser, err := cfg.db.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get user for refresh token", err)
		return
	}
	accessToken, err := auth.MakeJWT(
		dbUser.ID,
		cfg.secret,
		time.Hour,
	)

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate token", err)
		return
	}

	respondWithJson(w, http.StatusOK, response{
		Token: accessToken,
	})
	
}


func (cfg* apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find token", err)
		return
	}

	err = cfg.db.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't revoke session", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg* apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	var params parameters
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not decode parameters", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not find JWT", err)
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not validate JWT", err)
		return
	}
 
	hashedPasswd, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not hash the password", err)
		return
	}

	dbUser, err := cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{
		ID: userId,
		Email: params.Email,
		HashedPassword: hashedPasswd,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update the user email and password!", err)
		return
	}

	response := User {
		ID: dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email: dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
	}

	respondWithJson(w, http.StatusOK, response)

}

func (cfg* apiConfig) handlerUpgradeRed(w http.ResponseWriter, r *http.Request) {
	polkaKey, err := auth.GetAPIKey(r.Header)
	if polkaKey != cfg.polkaKey {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized webhook request!", err)
		return
	}

	type parameters struct {
		Event string `json:"event"`
		Data struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	
	var params parameters
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not decode parameters for webhooks", err)
		return
	}

	if params.Event != "user.upgraded" {
		respondWithError(w, http.StatusNoContent, "Not upgrading the user", err)
		return
	}

	userId, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Coiuld not convert parameters user ID to uuid", err)
		return

	}

	err = cfg.db.UpgradeRed(r.Context(), userId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User to upgrade not found in the database!", err)
		return
	}
	  
	w.WriteHeader(http.StatusNoContent)

}
