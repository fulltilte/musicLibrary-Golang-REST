package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type Song struct {
	ID          int    `db:"id"`
	GroupName   string `db:"group_name"`
	SongName    string `db:"song_name"`
	ReleaseDate string `db:"release_date"`
	SongText    string `db:"song_text"`
	Link        string `db:"link"`
}

type SongDetails struct {
	ReleaseDate string `json:"releaseDate"`
	SongText    string `json:"text"`
	Link        string `json:"link"`
}

func NewPostgresDB() (*sqlx.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_SSLMODE"))

	logrus.Debugf("Подключение к базе данных с параметрами: %s", connStr)

	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		logrus.WithError(err).Error("Ошибка подключения к базе данных")
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	if err := db.Ping(); err != nil {
		logrus.WithError(err).Error("Ошибка проверки подключения к базе данных (ping)")
		return nil, fmt.Errorf("ошибка ping в БД: %w", err)
	}

	logrus.Info("Успешное подключение к базе данных")
	return db, nil
}

func GetAllSongs(db *sqlx.DB, group, song string, limit, offset int) ([]Song, error) {
	var songs []Song

	query := "SELECT * FROM songs WHERE group_name ILIKE $1 AND song_name ILIKE $2 LIMIT $3 OFFSET $4"
	logrus.Debugf("Выполнение SQL-запроса: %s с параметрами group: %s, song: %s, limit: %d, offset: %d", query, group, song, limit, offset)

	err := db.Select(&songs, query, "%"+group+"%", "%"+song+"%", limit, offset)
	if err != nil {
		logrus.WithError(err).Error("Ошибка получения всех песен")
		return nil, err
	}

	logrus.Infof("Успешно получено: %d", len(songs))
	return songs, nil
}

func AddSong(db *sqlx.DB, song Song) error {
	query := "INSERT INTO songs (group_name, song_name, release_date, song_text, link) VALUES ($1, $2, $3, $4, $5)"
	logrus.Debugf("Выполнение SQL-запроса на добавление песни: %s, параметры: %v", query, song)

	_, err := db.Exec(query, song.GroupName, song.SongName, song.ReleaseDate, song.SongText, song.Link)
	if err != nil {
		logrus.WithError(err).Error("Ошибка добавления песни в базу данных")
		return err
	}

	logrus.Infof("Песня %s - %s успешно добавлена", song.GroupName, song.SongName)
	return nil
}

func GetSongText(db *sqlx.DB, id int) (string, error) {
	var text string
	query := "SELECT song_text FROM songs WHERE id = $1"
	logrus.Debugf("Выполнение SQL-запроса для получения текста песни с ID: %d", id)

	err := db.Get(&text, query, id)
	if err != nil {
		logrus.WithError(err).Errorf("Ошибка получения текста песни с ID: %d", id)
		return "", err
	}

	logrus.Infof("Текст песни с ID %d успешно получен", id)
	return text, nil
}

func GetSongByID(db *sqlx.DB, id int) (*Song, error) {
	var song Song
	query := "SELECT * FROM songs WHERE id = $1"
	logrus.Debugf("Выполнение SQL-запроса для получения песни по ID: %d", id)

	err := db.Get(&song, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logrus.Warnf("Песня с ID %d не найдена", id)
			return nil, nil
		}
		logrus.WithError(err).Errorf("Ошибка получения песни с ID: %d", id)
		return nil, err
	}

	logrus.Infof("Песня с ID %d успешно получена", id)
	return &song, nil
}

func UpdateSong(db *sqlx.DB, id int, song Song) error {
	query := `UPDATE songs SET group_name = $1, song_name = $2, release_date = $3, song_text = $4, link = $5 WHERE id = $6`
	logrus.Debugf("Выполнение SQL-запроса для обновления песни с ID: %d, параметры: %v", id, song)

	_, err := db.Exec(query, song.GroupName, song.SongName, song.ReleaseDate, song.SongText, song.Link, id)
	if err != nil {
		logrus.WithError(err).Errorf("Ошибка обновления песни с ID: %d", id)
		return err
	}

	logrus.Infof("Песня с ID %d успешно обновлена", id)
	return nil
}

func DeleteSong(db *sqlx.DB, id int) error {
	query := "DELETE FROM songs WHERE id = $1"
	logrus.Debugf("Выполнение SQL-запроса для удаления песни с ID: %d", id)

	_, err := db.Exec(query, id)
	if err != nil {
		logrus.WithError(err).Errorf("Ошибка удаления песни с ID: %d", id)
		return err
	}

	logrus.Infof("Песня с ID %d успешно удалена", id)
	return nil
}

func FetchSongDetails(group, song string) (SongDetails, error) {
	apiUrl := os.Getenv("API_URL")
	if apiUrl == "" {
		logrus.Error("API_URL не задан в env файле")
		return SongDetails{}, errors.New("API_URL не задан в env файле")
	}

	url := fmt.Sprintf("%s/info?group=%s&song=%s", apiUrl, group, song)
	logrus.Debugf("Запрос данных из внешнего API по URL: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		logrus.WithError(err).Error("Ошибка при запросе к API")
		return SongDetails{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("Ошибка получения данных из API, код ответа: %d", resp.StatusCode)
		return SongDetails{}, errors.New("ошибка получения данных из API")
	}

	var songDetails SongDetails
	if err := json.NewDecoder(resp.Body).Decode(&songDetails); err != nil {
		logrus.WithError(err).Error("Ошибка декодирования ответа от API")
		return SongDetails{}, err
	}

	logrus.Infof("Успешно получены данные для песни: %s - %s", group, song)
	return songDetails, nil
}
