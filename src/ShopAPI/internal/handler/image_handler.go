package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ImageHandler struct {
	imageService ImageService
}

func NewImageHandler(imageService ImageService) *ImageHandler {
	return &ImageHandler{imageService: imageService}
}

// UploadImage godoc
// @Summary      Загрузить изображение для товара
// @Tags         images
// @Accept       multipart/form-data
// @Param        product_id  formData  string  true  "ID товара"
// @Param        image       formData  file    true  "Файл изображения"
// @Success      201  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/images [post]
func (h *ImageHandler) UploadImage(c *gin.Context) {
	productIDStr := c.PostForm("product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный product_id"})
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "файл изображения обязателен"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не удалось открыть файл"})
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не удалось прочитать файл"})
		return
	}

	imageID, err := h.imageService.UploadImage(c.Request.Context(), productID, data)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": imageID})
}

// GetImageByID godoc
// @Summary      Получить изображение по ID
// @Tags         images
// @Produce      application/octet-stream
// @Param        id   path  string  true  "ID изображения"
// @Success      200  {file}  binary
// @Failure      404  {object}  map[string]string
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/images/{id} [get]
func (h *ImageHandler) GetImageByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	data, err := h.imageService.GetImageByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Header("Content-Disposition", "attachment")
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// GetImageByProductID godoc
// @Summary      Получить изображение товара
// @Tags         images
// @Produce      application/octet-stream
// @Param        product_id   path  string  true  "ID товара"
// @Success      200  {file}  binary
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/images/product/{product_id} [get]
func (h *ImageHandler) GetImageByProductID(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	data, err := h.imageService.GetImageByProductID(c.Request.Context(), productID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Header("Content-Disposition", "attachment")
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// UpdateImage godoc
// @Summary      Заменить изображение
// @Tags         images
// @Accept       multipart/form-data
// @Param        id     path      string  true  "ID изображения"
// @Param        image  formData  file    true  "Новый файл"
// @Success      204
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/images/{id} [put]
func (h *ImageHandler) UpdateImage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "файл изображения обязателен"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не удалось открыть файл"})
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не удалось прочитать файл"})
		return
	}

	if err := h.imageService.UpdateImage(c.Request.Context(), id, data); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// DeleteImage godoc
// @Summary      Удалить изображение
// @Tags         images
// @Param        id   path  string  true  "ID изображения"
// @Success      204
// @Security     BearerAuth
// @Failure      401  {object} map[string]string
// @Router       /api/v1/images/{id} [delete]
func (h *ImageHandler) DeleteImage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный UUID"})
		return
	}

	if err := h.imageService.DeleteImage(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
