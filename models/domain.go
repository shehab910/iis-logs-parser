package models

import (
	"iis-logs-parser/utils"

	"gorm.io/gorm"
)

type Domain struct {
	gorm.Model
	UserID uint `gorm:"not null"  validate:"required"` // Foreign Key to link to user

	DomainName     string `json:"domainName" gorm:"type:varchar(255);not null" validate:"required"` // Domain name
	Description    string `json:"description" gorm:"type:text"  validate:"required"`                // Description
	IsActive       bool   `json:"isActive" gorm:"default:true"`                                     // Boolean for active status
	BusinessField  string `json:"businessField" gorm:"type:varchar(255)"  validate:"required"`      // Business field
	BusinessSector string `json:"businessSector" gorm:"type:varchar(255)"  validate:"required"`     // Business sector
	IsSubdomain    bool   `json:"isSubdomain" gorm:"default:false"`                                 // Boolean for subdomain status
	Default        bool   `json:"default" gorm:"default:false"`                                     // Boolean for default domain status
}
type DomainUpdateRequest struct {
	DomainName     *string `json:"domainName,omitempty"`
	Description    *string `json:"description,omitempty"`
	IsActive       *bool   `json:"isActive,omitempty"`
	BusinessField  *string `json:"businessField,omitempty"`
	BusinessSector *string `json:"businessSector,omitempty"`
	IsSubdomain    *bool   `json:"isSubdomain,omitempty"`
	Default        *bool   `json:"default,omitempty"`
}

func (d *Domain) Validate() error {
	return utils.Validate.Struct(d)
}

func (d *Domain) PartialUpdate(update DomainUpdateRequest) {
	if update.DomainName != nil {
		d.DomainName = *update.DomainName
	}
	if update.Description != nil {
		d.Description = *update.Description
	}
	if update.IsActive != nil {
		d.IsActive = *update.IsActive
	}
	if update.BusinessField != nil {
		d.BusinessField = *update.BusinessField
	}
	if update.BusinessSector != nil {
		d.BusinessSector = *update.BusinessSector
	}
	if update.IsSubdomain != nil {
		d.IsSubdomain = *update.IsSubdomain
	}
	if update.Default != nil {
		d.Default = *update.Default
	}
}
