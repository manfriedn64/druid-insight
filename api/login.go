package api

import (
	"druid-insight/auth"
	"druid-insight/logging"
	"encoding/json"
	"net/http"
)

func LoginHandler(cfg *auth.Config, users *auth.UsersFile, loginLogger *logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			loginLogger.Write("LOGIN FAIL (bad json) user=" + req.Username)
			return
		}
		username := req.Username
		var userHash, userSalt string
		isAdmin := false

		if cfg.Auth.UserBackend == "file" {
			u, ok := users.Users[username]
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				loginLogger.Write("LOGIN FAIL (no user) user=" + username)
				return
			}
			userHash, userSalt = u.Hash, u.Salt
			isAdmin = u.Admin

			passHash, _ := auth.ApplyHashMacro(cfg.Auth.HashMacro, req.Password, username, userSalt, cfg.Auth.Salt)
			if passHash != userHash {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				loginLogger.Write("LOGIN FAIL (wrong pass) user=" + username)
				return
			}
		}
		// Ajoute ici la branche DB si tu veux
		tokenString, err := auth.GenerateJWT(cfg.JWT.Secret, username, isAdmin, cfg.JWT.ExpirationMinutes)
		if err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			loginLogger.Write("LOGIN FAIL (jwt error) user=" + username)
			return
		}
		resp := map[string]string{"token": tokenString}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		loginLogger.Write("LOGIN OK user=" + username)
	}
}
