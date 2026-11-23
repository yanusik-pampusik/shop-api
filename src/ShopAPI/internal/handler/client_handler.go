package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shopapi/internal/dto"
)

type ClientHandler struct {
	clientService ClientService
}

func NewClientHandler(clientService ClientService) *ClientHandler {
	return &ClientHandler{clientService: clientService}
}

// CreateClient godoc
// @Summary      Создать клиента
// @Description  Добавляет нового клиента в магазин
// @Tags         clients
// @Accept       json
// @Produce      json
// @Param        client  body      dto.ClientCreateDTO  true  "Данные клиента"
// @Success      201     {object}  dto.ClientDTO
// @Failure      400     {object}  map[string]string
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/clients [post]
func (h *ClientHandler) CreateClient(c *gin.Context) {
	var input dto.ClientCreateDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := h.clientService.CreateClient(c.Request.Context(), input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, client)
}

// GetClients godoc
// @Summary      Получить список клиентов
// @Description  Возвращает всех клиентов с опциональной пагинацией
// @Tags         clients
// @Produce      json
// @Param        limit   query     int  false  "Лимит записей"
// @Param        offset  query     int  false  "Смещение"
// @Success      200     {array}   dto.ClientDTO
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/clients [get]
func (h *ClientHandler) GetClients(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	clients, err := h.clientService.GetAllClients(c.Request.Context(), limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, clients)
}

// GetClientsByName godoc
// @Summary      Найти клиентов по имени и фамилии
// @Tags         clients
// @Produce      json
// @Param        name     query     string  true  "Имя"
// @Param        surname  query     string  true  "Фамилия"
// @Success      200      {array}   dto.ClientDTO
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/clients/search [get]
func (h *ClientHandler) GetClientsByName(c *gin.Context) {
	name := c.Query("name")
	surname := c.Query("surname")
	if name == "" || surname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "параметры name и surname обязательны"})
		return
	}

	clients, err := h.clientService.GetClientsByNameAndSurname(c.Request.Context(), name, surname)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, clients)
}

// GetClientByID godoc
// @Summary      Получить клиента по ID
// @Tags         clients
// @Produce      json
// @Param        id   path      string  true  "ID клиента"
// @Success      200  {object}  dto.ClientDTO
// @Failure      404  {object}  map[string]string
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/clients/{id} [get]
func (h *ClientHandler) GetClientByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	client, err := h.clientService.GetClientByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, client)
}

// UpdateClientAddress godoc
// @Summary      Изменить адрес клиента
// @Tags         clients
// @Accept       json
// @Produce      json
// @Param        id       path      string                       true  "ID клиента"
// @Param        address  body      dto.ClientUpdateAddressDTO   true  "Новый адрес"
// @Success      204
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/clients/{id}/address [patch]
func (h *ClientHandler) UpdateClientAddress(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	var input dto.ClientUpdateAddressDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clientService.UpdateClientAddress(c.Request.Context(), id, input); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// DeleteClient godoc
// @Summary      Удалить клиента
// @Tags         clients
// @Param        id   path  string  true  "ID клиента"
// @Success      204
// @Failure      404  {object}  map[string]string
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/clients/{id} [delete]
func (h *ClientHandler) DeleteClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	if err := h.clientService.DeleteClient(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
