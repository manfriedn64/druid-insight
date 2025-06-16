package utils

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

func PromptPasswordTwice() (string, error) {
	for {
		fmt.Print("Enter password: ")
		pass1, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", err
		}
		if len(pass1) < 8 {
			fmt.Println("Password must be at least 8 characters.")
			continue
		}
		fmt.Print("Repeat password: ")
		pass2, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", err
		}
		if string(pass1) != string(pass2) {
			fmt.Println("Passwords do not match. Try again.")
			continue
		}
		return string(pass1), nil
	}
}
