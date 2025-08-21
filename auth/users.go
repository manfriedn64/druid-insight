package auth

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"druid-insight/utils"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Listen        string            `yaml:"listen"`
		Static        string            `yaml:"static"`
		StaticDefault string            `yaml:"static_default"`
		StaticAllowed []string          `yaml:"static_allowed"`
		LogDir        string            `yaml:"log_dir"`
		TemplateVars  map[string]string `yaml:"template_vars"`
	} `yaml:"server"`
	JWT struct {
		Secret            string `yaml:"secret"`
		ExpirationMinutes int    `yaml:"expiration_minutes"`
	} `yaml:"jwt"`
	Auth struct {
		UserBackend string `yaml:"user_backend"` // "file", "mysql", "postgres", "sqlite"
		UserFile    string `yaml:"user_file"`
		HashMacro   string `yaml:"hash_macro"`
		Salt        string `yaml:"salt"`
		DBDSN       string `yaml:"db_dsn"`
		UserRequest string `yaml:"user_request"` // ex: SELECT hash, salt, is_admin FROM users WHERE name = ? AND pass = ?
		DBHashMacro string `yaml:"db_hash_macro"`
		DBPassHash  bool   `yaml:"db_pass_hash"`
	} `yaml:"auth"`
	Context         map[string]string `yaml:"context"`            // contexte global pour les requêtes Druid{
	MaxFileAgeHours int               `yaml:"max_file_age_hours"` // durée max en heures
}

type UsersFile struct {
	Users map[string]UserInfo `yaml:"users"`
}

type UserInfo struct {
	Hash   string                         `yaml:"hash"`
	Salt   string                         `yaml:"salt"`
	Admin  bool                           `yaml:"admin"`
	Access map[string]map[string][]string `yaml:"access,omitempty"` // si tu as ajouté la partie droits
}

func LoadConfig(file string) (*Config, error) {
	var cfg Config
	root := utils.GetProjectRoot()
	cfgPath := filepath.Join(root, file)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func LoadUsers(file string) (*UsersFile, error) {
	var uf UsersFile
	root := utils.GetProjectRoot()
	cfgPath := filepath.Join(root, file)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &uf); err != nil {
		return nil, err
	}
	return &uf, nil
}

// Ex: "SELECT hash, salt, is_admin FROM users WHERE name = ? AND password =  ? "
func GetUserFromDB(db *sql.DB, query, username string, password string) (hash, salt string, isAdmin bool, err error) {
	row := db.QueryRow(query, username, password)
	var adminVal interface{}
	err = row.Scan(&hash, &salt, &adminVal)
	if err != nil {
		log.Println(err)
		return "", "", false, err
	}
	isAdmin = dbToBool(adminVal)
	return
}

func dbToBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int64:
		return val != 0
	case int:
		return val != 0
	case []uint8:
		s := string(val)
		return s == "1" || s == "t" || s == "T" || s == "true" || s == "TRUE"
	}
	return false
}

func ApplyHashMacro(macro, password, user, userSalt, globalSalt string) (string, error) {
	replace := func(s string) string {
		s = strings.ReplaceAll(s, "{password}", password)
		s = strings.ReplaceAll(s, "{user}", user)
		s = strings.ReplaceAll(s, "{salt}", userSalt)
		s = strings.ReplaceAll(s, "{globalsalt}", globalSalt)
		return s
	}
	macro = strings.TrimSpace(macro)
	if strings.HasPrefix(macro, "{sha256}") {
		plain := extractBetween(macro, "{sha256}(", ")")
		plain = replace(plain)
		return sha256Hash(plain), nil
	}
	if strings.HasPrefix(macro, "{sha1}") {
		plain := extractBetween(macro, "{sha1}(", ")")
		plain = replace(plain)
		return sha1Hash(plain), nil
	}
	if strings.HasPrefix(macro, "{md5}") {
		plain := extractBetween(macro, "{md5}(", ")")
		plain = replace(plain)
		return md5Hash(plain), nil
	}
	if strings.HasPrefix(macro, "{clear}") {
		plain := extractBetween(macro, "{clear}(", ")")
		plain = replace(plain)
		return plain, nil
	}
	return "", errors.New("unsupported hash macro")
}

func extractBetween(str, start, end string) string {
	a := strings.Index(str, start)
	if a == -1 {
		return ""
	}
	a += len(start)
	b := strings.LastIndex(str, end)
	if b == -1 || b <= a {
		return ""
	}
	return str[a:b]
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
func sha1Hash(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
func md5Hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
