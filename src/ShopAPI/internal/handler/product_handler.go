package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shopapi/internal/dto"
)

type ProductHandler struct {
	productService ProductService
}

func NewProductHandler(productService ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

// CreateProduct godoc
// @Summary      Создать товар
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        product  body      dto.ProductCreateDTO  true  "Данные товара"
// @Success      201      {object}  dto.ProductDTO
// @Failure      400      {object}  map[string]string
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var input dto.ProductCreateDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.productService.CreateProduct(c.Request.Context(), input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, product)
}

// GetProducts godoc
// @Summary      Получить все товары
// @Tags         products
// @Produce      json
// @Success      200  {array}   dto.ProductDTO
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/products [get]
func (h *ProductHandler) GetProducts(c *gin.Context) {
	products, err := h.productService.GetAllProducts(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, products)
}

// GetProductByID godoc
// @Summary      Получить товар по ID
// @Tags         products
// @Produce      json
// @Param        id   path      string  true  "ID товара"
// @Success      200  {object}  dto.ProductDTO
// @Failure      404  {object}  map[string]string
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/products/{id} [get]
func (h *ProductHandler) GetProductByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	product, err := h.productService.GetProductByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

// DecreaseStock godoc
// @Summary      Уменьшить количество товара
// @Tags         products
// @Accept       json
// @Param        id       path      string                      true  "ID товара"
// @Param        amount   body      dto.ProductDecreaseStockDTO true  "Сколько уменьшить"
// @Success      204
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/products/{id}/decrease [patch]
func (h *ProductHandler) DecreaseStock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	var input dto.ProductDecreaseStockDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.productService.DecreaseStock(c.Request.Context(), id, input.Amount); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// DeleteProduct godoc
// @Summary      Удалить товар
// @Tags         products
// @Param        id   path  string  true  "ID товара"
// @Success      204
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	if err := h.productService.DeleteProduct(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
