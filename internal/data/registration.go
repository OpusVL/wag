package data

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/NHAS/wag/pkg/control"
)

func GetRegistrationToken(token string) (username, overwrites string, group []string, err error) {

	minTime := time.After(1 * time.Second)

	var groupsJson sql.NullString

	err = database.QueryRow(`
		SELECT 
			token, username, overwrite, groups 
		FROM 
			RegistrationTokens
		WHERE
			token = ?
	`, token).Scan(&token, &username, &overwrites, &groupsJson)
	if err != nil {
		return
	}

	if groupsJson.Valid {
		err = json.Unmarshal([]byte(groupsJson.String), &group)
	}

	<-minTime

	return
}

// Returns list of tokens
func GetRegistrationTokens() (result []control.RegistrationResult, err error) {

	rows, err := database.Query("SELECT token, username, overwrite, groups FROM RegistrationTokens ORDER by ROWID DESC")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var (
			groupsJson   sql.NullString
			registration control.RegistrationResult
		)
		err = rows.Scan(&registration.Token, &registration.Username, &registration.Overwrites, &groupsJson)
		if err != nil {
			return nil, err
		}

		if groupsJson.Valid {
			err = json.Unmarshal([]byte(groupsJson.String), &registration.Groups)
			if err != nil {
				return
			}
		}

		result = append(result, registration)
	}

	return result, nil
}

func DeleteRegistrationToken(identifier string) error {
	_, err := database.Exec(`
		DELETE FROM
			RegistrationTokens
		WHERE
			token = $1 OR username = $1
	`, identifier)
	return err
}

// Randomly generate a token for a specific username
func GenerateToken(username, overwrite string, groups []string) (token string, err error) {
	tokenBytes, err := generateRandomBytes(32)
	if err != nil {
		return "", err
	}

	token = hex.EncodeToString(tokenBytes)
	err = AddRegistrationToken(token, username, overwrite, groups)

	return
}

// Add a token to the database to add or overwrite a device for a user, may fail of the token does not meet complexity requirements
func AddRegistrationToken(token, username, overwrite string, groups []string) error {
	if len(token) < 32 {
		return errors.New("Registration token is too short")
	}

	if !allowedTokenCharacters.Match([]byte(token)) {
		return errors.New("Registration token contains illegal characters (allowed characters a-z A-Z - . _ )")
	}

	var err error
	if overwrite != "" {
		var u string
		err = database.QueryRow("SELECT address FROM Devices WHERE address = ? AND username = ?", overwrite, username).Scan(&u)
		if err != nil {
			if err != sql.ErrNoRows {
				return errors.New("could not find device that this token is intended to overwrite")
			}
			return errors.New("failed to create registration token: " + err.Error())
		}
	}

	if len(groups) != 0 {

		result, _ := json.Marshal(groups)

		_, err = database.Exec(`
		INSERT INTO
			RegistrationTokens (token, username, overwrite, groups)
		VALUES
			(?, ?, ?, ?)
	`, token, username, overwrite, string(result))

		return err
	}

	_, err = database.Exec(`
	INSERT INTO
		RegistrationTokens (token, username, overwrite)
	VALUES
		(?, ?, ?)
`, token, username, overwrite)

	return err
}

func generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}