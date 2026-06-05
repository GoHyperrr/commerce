package seo

// SEO represents generic search engine optimization metadata that can be
// embedded in other models (such as products, taxonomy terms, or pages).
type SEO struct {
	MetaTitle       string `gorm:"column:meta_title" json:"metaTitle"`
	MetaDescription string `gorm:"column:meta_description" json:"metaDescription"`
	MetaKeywords    string `gorm:"column:meta_keywords" json:"metaKeywords"`
	MetaImage       string `gorm:"column:meta_image" json:"metaImage"`
}
