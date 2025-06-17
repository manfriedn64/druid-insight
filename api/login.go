package api

import (
	"database/sql"
	"druid-insight/auth"
	"druid-insight/logging"
	"encoding/json"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
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
			log.Println("LOGIN FAIL (bad json) user=" + req.Username)
			return
		}
		username := req.Username
		var userHash, userSalt string
		isAdmin := false

		if cfg.Auth.UserBackend == "file" {
			u, ok := users.Users[username]
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				log.Println("LOGIN FAIL (no user) user=" + username)
				return
			}
			userHash, userSalt = u.Hash, u.Salt
			isAdmin = u.Admin

			passHash, _ := auth.ApplyHashMacro(cfg.Auth.HashMacro, req.Password, username, userSalt, cfg.Auth.Salt)
			if passHash != userHash {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				log.Println("LOGIN FAIL (wrong pass) user=" + username)
				return
			}
		} else if cfg.Auth.UserBackend == "mysql" || cfg.Auth.UserBackend == "postgres" || cfg.Auth.UserBackend == "sqlite" {
			db, err := sql.Open(cfg.Auth.UserBackend, cfg.Auth.DBDSN)
			if err != nil {
				http.Error(w, "Erreur base de données", http.StatusInternalServerError)
				log.Println("LOGIN FAIL (db open) user=" + username)
				return
			}
			defer db.Close()

			userHash, userSalt, isAdmin, err = auth.GetUserFromDB(db, cfg.Auth.UserRequest, username, req.Password)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				log.Println("LOGIN FAIL (db no user) user=" + username)
				return
			}
			// if DBPassHash is true, it means the password hash was not checked
			// by db sql call above, so it needs to be done now
			if cfg.Auth.DBPassHash {
				passHash, _ := auth.ApplyHashMacro(cfg.Auth.DBHashMacro, req.Password, username, userSalt, cfg.Auth.Salt)
				if passHash != userHash {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					log.Println("LOGIN FAIL (db wrong pass) user=" + username)
					return
				}
			} else {
				log.Println("LOGIN supposedly validate by DB")
			}
		}
		// Ajoute ici la branche DB si tu veux
		tokenString, err := auth.GenerateJWT(cfg.JWT.Secret, username, isAdmin, cfg.JWT.ExpirationMinutes)
		if err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			log.Println("LOGIN FAIL (jwt error) user=" + username)
			return
		}
		resp := map[string]string{"token": tokenString}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		log.Println("LOGIN OK user=" + username)
	}
}
