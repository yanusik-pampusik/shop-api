package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shopapi/internal/dto"
)

type SupplierHandler struct {
	supplierService SupplierService
}

func NewSupplierHandler(supplierService SupplierService) *SupplierHandler {
	return &SupplierHandler{supplierService: supplierService}
}

// CreateSupplier godoc
// @Summary      Создать поставщика
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        supplier  body      dto.SupplierCreateDTO  true  "Данные поставщика"
// @Success      201       {object}  dto.SupplierDTO
// @Failure      400       {object}  map[string]string
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/suppliers [post]
func (h *SupplierHandler) CreateSupplier(c *gin.Context) {
	var input dto.SupplierCreateDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	supplier, err := h.supplierService.CreateSupplier(c.Request.Context(), input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, supplier)
}

// GetSuppliers godoc
// @Summary      Получить всех поставщиков
// @Tags         suppliers
// @Produce      json
// @Success      200  {array}   dto.SupplierDTO
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/suppliers [get]
func (h *SupplierHandler) GetSuppliers(c *gin.Context) {
	suppliers, err := h.supplierService.GetAllSuppliers(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, suppliers)
}

// GetSupplierByID godoc
// @Summary      Получить поставщика по ID
// @Tags         suppliers
// @Produce      json
// @Param        id   path      string  true  "ID поставщика"
// @Success      200  {object}  dto.SupplierDTO
// @Failure      404  {object}  map[string]string
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/suppliers/{id} [get]
func (h *SupplierHandler) GetSupplierByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	supplier, err := h.supplierService.GetSupplierByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, supplier)
}

// UpdateSupplierAddress godoc
// @Summary      Изменить адрес поставщика
// @Tags         suppliers
// @Accept       json
// @Param        id       path      string                         true  "ID поставщика"
// @Param        address  body      dto.SupplierUpdateAddressDTO   true  "Новый адрес"
// @Success      204
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/suppliers/{id}/address [patch]
func (h *SupplierHandler) UpdateSupplierAddress(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	var input dto.SupplierUpdateAddressDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.supplierService.UpdateSupplierAddress(c.Request.Context(), id, input); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// DeleteSupplier godoc
// @Summary      Удалить поставщика
// @Tags         suppliers
// @Param        id   path  string  true  "ID поставщика"
// @Success      204
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/suppliers/{id} [delete]
func (h *SupplierHandler) DeleteSupplier(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	if err := h.supplierService.DeleteSupplier(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
