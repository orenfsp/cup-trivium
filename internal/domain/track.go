package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// trackAlphabet — алфавит без похожих символов (0, O, 1, I, l).
const trackAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// GenerateTrackNumber создаёт трек-номер вида ОТК-X7KD-R9MF-Q3HP
// (12 значащих символов) криптографически стойким генератором.
func GenerateTrackNumber() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	chars := make([]byte, 12)
	for i, b := range buf {
		chars[i] = trackAlphabet[int(b)%len(trackAlphabet)]
	}
	return fmt.Sprintf("ОТК-%s-%s-%s", string(chars[0:4]), string(chars[4:8]), string(chars[8:12])), nil
}

// NormalizeTrack приводит трек-номер к каноническому виду.
func NormalizeTrack(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)
	s = strings.ReplaceAll(s, "Ё", "Е")
	s = strings.ReplaceAll(s, "OTK", "ОТК")
	parts := strings.Split(s, "-")
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}

// HashTrack считает SHA-256 от нормализованного номера.
// В БД хранится только хеш: трек-номер является bearer credential.
func HashTrack(number string) string {
	h := sha256.Sum256([]byte(NormalizeTrack(number)))
	return hex.EncodeToString(h[:])
}
