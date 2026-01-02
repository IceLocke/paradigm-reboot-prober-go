package util

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"paradigm-reboot-prober-go/internal/model"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// GenerateCSV generates a CSV string from a list of SongLevelWithScore records
func GenerateCSV(records []model.SongLevelWithScore) (string, error) {
	var buf bytes.Buffer
	// Write UTF-8 BOM for Excel compatibility
	buf.Write([]byte("\ufeff"))

	writer := csv.NewWriter(&buf)
	header := []string{"song_level_id", "title", "version", "difficulty", "level", "score"}
	if err := writer.Write(header); err != nil {
		return "", err
	}

	for _, rec := range records {
		row := []string{
			strconv.Itoa(rec.SongLevelID),
			rec.Title,
			rec.Version,
			string(rec.Difficulty),
			strconv.FormatFloat(rec.Level, 'f', 1, 64),
			strconv.Itoa(rec.Score),
		}
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}

	writer.Flush()
	return buf.String(), writer.Error()
}

// decodeToUTF8 attempts to convert content to UTF-8 if it's not already.
// It specifically handles GBK which is common on Windows.
func decodeToUTF8(content []byte) ([]byte, error) {
	if utf8.Valid(content) {
		return content, nil
	}

	// Try GBK to UTF-8
	reader := transform.NewReader(bytes.NewReader(content), simplifiedchinese.GBK.NewDecoder())
	return io.ReadAll(reader)
}

// GetRecordsFromCSV parses a CSV file into a list of PlayRecordBase
func GetRecordsFromCSV(filePath string) ([]model.PlayRecordBase, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read the whole file to handle BOM and encoding easily
	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	// Detect and convert encoding
	content, err = decodeToUTF8(content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file encoding: %v", err)
	}

	// Remove UTF-8 BOM if present
	sContent := string(content)
	sContent = strings.TrimPrefix(sContent, "\ufeff")

	reader := csv.NewReader(strings.NewReader(sContent))
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	// Create a map for header indices
	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	var records []model.PlayRecordBase
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip malformed rows
		}

		record := model.PlayRecordBase{}

		if idx, ok := headerMap["song_level_id"]; ok && idx < len(row) {
			id, _ := strconv.Atoi(row[idx])
			record.SongLevelID = id
		}

		if idx, ok := headerMap["score"]; ok && idx < len(row) {
			score, _ := strconv.Atoi(row[idx])
			record.Score = score
		}

		if record.SongLevelID != 0 {
			records = append(records, record)
		}
	}

	return records, nil
}

// GenerateEmptyCSV creates a default CSV file with headers and song level info
func GenerateEmptyCSV(filePath string, songLevels []model.SongLevel) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write UTF-8 BOM
	f.Write([]byte("\ufeff"))

	writer := csv.NewWriter(f)
	header := []string{"song_level_id", "title", "version", "difficulty", "level", "score"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, sl := range songLevels {
		title := ""
		version := ""
		if sl.Song != nil {
			title = sl.Song.Title
			version = sl.Song.Version
		}

		row := []string{
			strconv.Itoa(sl.SongLevelID),
			title,
			version,
			string(sl.Difficulty),
			strconv.FormatFloat(sl.Level, 'f', 1, 64),
			"0",
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}
