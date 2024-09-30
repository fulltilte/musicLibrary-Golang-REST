package handler

import (
	"musicLibrary-Golang-REST/internal/models"
	"musicLibrary-Golang-REST/internal/repository"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	db *sqlx.DB
}

func NewHandler(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) InitRoutes(router *gin.Engine) {
	router.GET("/songs", h.getSongs)
	router.GET("/songs/:id/text", h.getSongText)
	router.POST("/songs", h.addSong)
	router.PUT("/songs/:id", h.updateSong)
	router.DELETE("/songs/:id", h.deleteSong)
}

// @Summary Получение списка песен
// @Description Возвращает список песен с возможностью фильтрации по группе и названию, а также с пагинацией
// @Tags Songs
// @Param group query string false "Фильтр по группе"
// @Param song query string false "Фильтр по названию песни"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество песен на странице" default(10)
// @Success 200 {array} models.Song "Список песен"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /songs [get]
func (h *Handler) getSongs(c *gin.Context) {
	group := c.Query("group")
	song := c.Query("song")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	offset := (page - 1) * limit

	logrus.Debugf("Получение песен с группой: %s, песней: %s, страница: %d, лимит: %d", group, song, page, limit)

	songs, err := repository.GetAllSongs(h.db, group, song, limit, offset)
	if err != nil {
		logrus.WithError(err).Error("Ошибка получения песен")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения песен"})
		return
	}

	logrus.Infof("Возвращено: %d", len(songs))
	c.JSON(http.StatusOK, songs)
}

// @Summary Получение текста песни
// @Description Возвращает текст песни по её ID с пагинацией по куплетам
// @Tags Songs
// @Param id path int true "ID песни"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество куплетов на странице" default(1)
// @Success 200 {object} map[string]interface{} "Куплеты песни с пагинацией"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /songs/{id}/text [get]
func (h *Handler) getSongText(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	logrus.Debugf("Получение текста песни по ID: %d", id)

	text, err := repository.GetSongText(h.db, id)
	if err != nil {
		logrus.WithError(err).Error("Ошибка получения текста песни")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения текста песни"})
		return
	}

	verses := splitVerses(text)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1"))

	start := (page - 1) * limit
	end := start + limit

	logrus.Debugf("Разбиение на страницы, страница: %d, лимит: %d", page, limit)

	if start > len(verses) {
		logrus.Info("Запрошенная страница превышает количество куплетов")
		c.JSON(http.StatusOK, gin.H{"verses": []string{}})
		return
	}

	if end > len(verses) {
		end = len(verses)
	}

	paginatedVerses := verses[start:end]
	logrus.Infof("Возвращено %d куплетов для страницы %d", len(paginatedVerses), page)
	c.JSON(http.StatusOK, gin.H{"verses": paginatedVerses, "page": page, "limit": limit})
}

// @Summary Добавление новой песни
// @Description Добавляет новую песню с информацией о группе и названии
// @Tags Songs
// @Accept  json
// @Produce  json
// @Param input body models.Song true "Данные новой песни"
// @Success 200 {object} map[string]interface{} "Сообщение об успешном добавлении"
// @Failure 400 {object} map[string]interface{} "Ошибка в данных запроса"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /songs [post]
func (h *Handler) addSong(c *gin.Context) {
	var input models.Song
	if err := c.BindJSON(&input); err != nil {
		logrus.WithError(err).Error("Ошибка привязки JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка с входными данными"})
		return
	}

	logrus.Infof("Добавление новой песни: %s - %s", input.GroupName, input.SongName)

	songDetails, err := repository.FetchSongDetails(input.GroupName, input.SongName)
	if err != nil {
		logrus.WithError(err).Error("Ошибка получения данных из внешнего API")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения данных из внешнего API"})
		return
	}

	newSong := repository.Song{
		GroupName:   input.GroupName,
		SongName:    input.SongName,
		ReleaseDate: songDetails.ReleaseDate,
		SongText:    songDetails.SongText,
		Link:        songDetails.Link,
	}

	err = repository.AddSong(h.db, newSong)
	if err != nil {
		logrus.WithError(err).Error("Ошибка добавления песни в базу данных")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка добавления песни в БД"})
		return
	}

	logrus.Infof("Песня %s - %s успешно добавлена", input.GroupName, input.SongName)
	c.JSON(http.StatusOK, gin.H{"message": "Песня успешно добавлена"})
}

// @Summary Обновление информации о песне
// @Description Обновляет информацию о песне по её ID
// @Tags Songs
// @Accept  json
// @Produce  json
// @Param id path int true "ID песни"
// @Param input body models.Song true "Новые данные песни"
// @Success 200 {object} map[string]interface{} "Сообщение об успешном обновлении"
// @Failure 400 {object} map[string]interface{} "Ошибка в данных запроса"
// @Failure 404 {object} map[string]interface{} "Песня не найдена"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /songs/{id} [put]
func (h *Handler) updateSong(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var input models.Song
	if err := c.BindJSON(&input); err != nil {
		logrus.WithError(err).Error("Ошибка привязки JSON для обновления")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка с входными данными"})
		return
	}

	logrus.Infof("Обновление песни ID: %d", id)

	currSong, err := repository.GetSongByID(h.db, id)
	if err != nil {
		logrus.WithError(err).Error("Ошибка получения данных песни для обновления")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения данных песни"})
		return
	}
	if currSong == nil {
		logrus.Warnf("Песня с ID %d не найдена", id)
		c.JSON(http.StatusNotFound, gin.H{"error": "Песня не найдена"})
		return
	}

	if input.GroupName != "" {
		currSong.GroupName = input.GroupName
	}
	if input.SongName != "" {
		currSong.SongName = input.SongName
	}
	if input.ReleaseDate != "" {
		currSong.ReleaseDate = input.ReleaseDate
	}
	if input.SongText != "" {
		currSong.SongText = input.SongText
	}
	if input.Link != "" {
		currSong.Link = input.Link
	}

	if err := repository.UpdateSong(h.db, id, *currSong); err != nil {
		logrus.WithError(err).Error("Ошибка обновления песни в базе данных")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления песни"})
		return
	}

	logrus.Infof("Песня ID %d успешно обновлена", id)
	c.JSON(http.StatusOK, gin.H{"message": "Песня успешно обновлена"})
}

// @Summary Удаление песни
// @Description Удаляет песню по её ID
// @Tags Songs
// @Param id path int true "ID песни"
// @Success 200 {object} map[string]interface{} "Сообщение об успешном удалении"
// @Failure 404 {object} map[string]interface{} "Песня не найдена"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /songs/{id} [delete]
func (h *Handler) deleteSong(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	logrus.Infof("Удаление песни ID: %d", id)

	if err := repository.DeleteSong(h.db, id); err != nil {
		logrus.WithError(err).Error("Ошибка удаления песни")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления песни"})
		return
	}

	logrus.Infof("Песня ID %d успешно удалена", id)
	c.JSON(http.StatusOK, gin.H{"message": "Песня успешно удалена"})
}

func splitVerses(text string) []string {
	return strings.Split(text, "\n\n")
}
