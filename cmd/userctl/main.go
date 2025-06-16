package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"druid-insight/auth"
	"druid-insight/utils"

	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	cmd := os.Args[1]
	switch cmd {
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Usage: userctl add <username>")
			os.Exit(1)
		}
		username := os.Args[2]
		addUser(username)
	case "disable":
		if len(os.Args) < 3 {
			fmt.Println("Usage: userctl disable <username>")
			os.Exit(1)
		}
		disableUser(os.Args[2])
	case "list":
		listUsers()
	default:
		usage()
	}
}

func usage() {
	fmt.Println(`Usage: userctl [add|disable|list] <username>

add <username>       : Ajoute un utilisateur interactif (mot de passe demandé)
disable <username>   : Commente/supprime un utilisateur (soft, dans users.yaml)
list                 : Liste tous les utilisateurs`)
}

// Demande un mot de passe à l’admin (masqué si possible)
func promptPassword() (string, error) {
	pass, err := utils.PromptPasswordTwice()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(pass), nil
}

func addUser(username string) {
	cfg, err := auth.LoadConfig("config.yaml")
	if err != nil {
		fmt.Println("Erreur lecture config.yaml :", err)
		os.Exit(1)
	}
	usersFile := cfg.Auth.UserFile
	users, err := auth.LoadUsers(usersFile)
	if err != nil {
		if os.IsNotExist(err) {
			users = &auth.UsersFile{Users: make(map[string]struct {
				Hash  string `yaml:"hash"`
				Salt  string `yaml:"salt"`
				Admin bool   `yaml:"admin"`
			})}
		} else {
			fmt.Println("Erreur lecture users.yaml :", err)
			os.Exit(1)
		}
	}
	if _, exists := users.Users[username]; exists {
		fmt.Println("L'utilisateur existe déjà.")
		os.Exit(1)
	}
	pass, err := promptPassword()
	if err != nil {
		fmt.Println("Erreur :", err)
		os.Exit(1)
	}
	salt := utils.RandomHex(8)
	hash, err := auth.ApplyHashMacro(cfg.Auth.HashMacro, pass, username, salt, cfg.Auth.Salt)
	if err != nil {
		fmt.Println("Erreur hashage :", err)
		os.Exit(1)
	}
	fmt.Print("Est-ce un administrateur ? (y/N) : ")
	admin := false
	var rep string
	fmt.Scanln(&rep)
	if rep == "y" || rep == "Y" || rep == "oui" || rep == "O" {
		admin = true
	}
	users.Users[username] = struct {
		Hash  string `yaml:"hash"`
		Salt  string `yaml:"salt"`
		Admin bool   `yaml:"admin"`
	}{Hash: hash, Salt: salt, Admin: admin}
	saveUsers(usersFile, users)
	fmt.Println("Utilisateur ajouté.")
}

func disableUser(username string) {
	cfg, err := auth.LoadConfig("config.yaml")
	if err != nil {
		fmt.Println("Erreur lecture config.yaml :", err)
		os.Exit(1)
	}
	usersFile := cfg.Auth.UserFile

	// Lire tout le users.yaml en texte
	lines, err := utils.ReadLines(filepath.Join(utils.GetProjectRoot(), "users.yaml"))
	if err != nil {
		fmt.Println("Erreur lecture users.yaml :", err)
		os.Exit(1)
	}
	out := []string{}
	inUser := false

	for _, l := range lines {
		trim := strings.TrimSpace(l)
		// Détection du début d'un user
		if strings.HasPrefix(trim, username+":") && !strings.HasPrefix(trim, "#") {
			inUser = true
			out = append(out, "# "+l)
			continue
		}
		if inUser {
			if strings.HasPrefix(trim, "#") {
				// déjà commenté
				out = append(out, l)
			} else if strings.HasPrefix(trim, "-") || (!strings.HasPrefix(trim, "") && !strings.HasPrefix(trim, "#")) && strings.HasSuffix(trim, ":") {
				// nouvelle entrée utilisateur
				inUser = false
				out = append(out, l)
			} else if trim == "" {
				inUser = false
				out = append(out, l)
			} else {
				out = append(out, "# "+l)
			}
			continue
		}
		out = append(out, l)
	}

	if !strings.Contains(strings.Join(out, "\n"), "# "+username+":") {
		fmt.Println("Utilisateur non trouvé ou déjà commenté.")
		return
	}

	err = os.WriteFile(usersFile, []byte(strings.Join(out, "\n")+"\n"), 0644)
	if err != nil {
		fmt.Println("Erreur écriture :", err)
		os.Exit(1)
	}
	fmt.Println("Utilisateur commenté/désactivé dans le YAML.")
}

func listUsers() {
	cfg, err := auth.LoadConfig("config.yaml")
	if err != nil {
		fmt.Println("Erreur lecture config.yaml :", err)
		os.Exit(1)
	}
	usersFile := cfg.Auth.UserFile
	users, err := auth.LoadUsers(usersFile)
	if err != nil {
		fmt.Println("Erreur lecture users.yaml :", err)
		os.Exit(1)
	}
	fmt.Println("Utilisateurs enregistrés :")
	for u, info := range users.Users {
		role := "user"
		if info.Admin {
			role = "admin"
		}
		fmt.Printf("- %s [%s]\n", u, role)
	}
}

func saveUsers(usersFile string, users *auth.UsersFile) {
	out, err := yaml.Marshal(users)
	if err != nil {
		fmt.Println("Erreur yaml :", err)
		os.Exit(1)
	}
	err = os.WriteFile(usersFile, out, 0644)
	if err != nil {
		fmt.Println("Erreur écriture :", err)
		os.Exit(1)
	}
}
