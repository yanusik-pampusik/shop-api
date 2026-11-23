package dto

import (
	"time"

	"shopapi/internal/models"
)

// --- Address ---

func AddressFromModel(a *models.Address) AddressDTO {
	return AddressDTO{
		ID:      a.ID,
		Country: a.Country,
		City:    a.City,
		Street:  a.Street,
	}
}

func AddressToModel(a *AddressCreateDTO) models.Address {
	return models.Address{
		Country: a.Country,
		City:    a.City,
		Street:  a.Street,
	}
}

// --- Category ---

func CategoryFromModel(c *models.Category) CategoryDTO {
	return CategoryDTO{
		ID:   c.ID,
		Name: c.Name,
	}
}

// --- Client ---

func ClientFromModel(c *models.Client) ClientDTO {
	return ClientDTO{
		ID:               c.ID,
		ClientName:       c.ClientName,
		ClientSurname:    c.ClientSurname,
		Birthday:         c.Birthday.Format("2006-01-02"),
		Gender:           c.Gender,
		RegistrationDate: c.RegistrationDate.Format(time.RFC3339),
		AddressID:        c.AddressID,
		Address:          AddressFromModel(&c.Address),
	}
}

func ClientToModel(c *ClientCreateDTO) (models.Client, error) {
	birthday, err := time.Parse("2006-01-02", c.Birthday)
	if err != nil {
		return models.Client{}, err
	}
	return models.Client{
		ClientName:    c.ClientName,
		ClientSurname: c.ClientSurname,
		Birthday:      birthday,
		Gender:        c.Gender,
		AddressID:     c.AddressID,
	}, nil
}

// --- Supplier ---

func SupplierFromModel(s *models.Supplier) SupplierDTO {
	return SupplierDTO{
		ID:          s.ID,
		Name:        s.Name,
		AddressID:   s.AddressID,
		Address:     AddressFromModel(&s.Address),
		PhoneNumber: s.PhoneNumber,
	}
}

func SupplierToModel(s *SupplierCreateDTO) models.Supplier {
	return models.Supplier{
		Name:        s.Name,
		AddressID:   s.AddressID,
		PhoneNumber: s.PhoneNumber,
	}
}

// --- Product ---

func ProductFromModel(p *models.Product) ProductDTO {
	dto := ProductDTO{
		ID:             p.ID,
		Name:           p.Name,
		CategoryID:     p.CategoryID,
		Category:       CategoryFromModel(&p.Category),
		Price:          p.Price,
		AvailableStock: p.AvailableStock,
		LastUpdateDate: p.LastUpdateDate.Format(time.RFC3339),
		SupplierID:     p.SupplierID,
		Supplier:       SupplierFromModel(&p.Supplier),
		ImageID:        p.ImageID,
	}
	return dto
}

func ProductToModel(p *ProductCreateDTO) models.Product {
	return models.Product{
		Name:           p.Name,
		CategoryID:     p.CategoryID,
		Price:          *p.Price,
		AvailableStock: p.AvailableStock,
		SupplierID:     p.SupplierID,
	}
}
