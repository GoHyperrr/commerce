package product

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoHyperrr/commerce/seo"
	"github.com/GoHyperrr/mdk"
)

// ValidateProduct checks if the product data is valid.
func (m *Module) ValidateProduct(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	productData, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing product input")
	}

	ba, err := json.Marshal(productData)
	if err != nil {
		return nil, fmt.Errorf("failed to process input: %w", err)
	}

	var in CreateProductInput
	if err := json.Unmarshal(ba, &in); err != nil {
		return nil, fmt.Errorf("failed to parse product input: %w", err)
	}

	if in.Name == "" {
		return nil, fmt.Errorf("product name is required")
	}

	if in.Handle == "" {
		return nil, fmt.Errorf("product handle is required")
	}

	for _, v := range in.Variants {
		if v.Title == "" {
			return nil, fmt.Errorf("variant title is required")
		}
		if v.Price < 0 {
			return nil, fmt.Errorf("variant price cannot be negative")
		}
	}

	return productData, nil
}

// PersistProduct saves the product to the database.
func (m *Module) PersistProduct(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	resRaw, ok := data["validate"]
	if !ok {
		return nil, fmt.Errorf("missing validated product data")
	}
	validatedData, ok := resRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid validated product data format")
	}

	ba, err := json.Marshal(validatedData)
	if err != nil {
		return nil, err
	}
	var in CreateProductInput
	if err := json.Unmarshal(ba, &in); err != nil {
		return nil, err
	}

	pDetails := ProductDetails{}
	if in.Details != nil {
		pDetails = ProductDetails{
			SKU:                      in.Details.SKU,
			Barcode:                  in.Details.Barcode,
			HSNCode:                  in.Details.HSNCode,
			Weight:                   in.Details.Weight,
			Length:                   in.Details.Length,
			Width:                    in.Details.Width,
			Height:                   in.Details.Height,
			FileUrl:                  in.Details.FileUrl,
			MaxDownloads:             in.Details.MaxDownloads,
			DownloadExpirationHours: in.Details.DownloadExpirationHours,
		}
	}

	pSEO := seo.SEO{}
	if in.SEO != nil {
		if in.SEO.MetaTitle != nil {
			pSEO.MetaTitle = *in.SEO.MetaTitle
		}
		if in.SEO.MetaDescription != nil {
			pSEO.MetaDescription = *in.SEO.MetaDescription
		}
		if in.SEO.MetaKeywords != nil {
			pSEO.MetaKeywords = *in.SEO.MetaKeywords
		}
		if in.SEO.MetaImage != nil {
			pSEO.MetaImage = *in.SEO.MetaImage
		}
	}

	pType := "PHYSICAL"
	if in.Type != nil {
		pType = *in.Type
	}
	pStatus := "DRAFT"
	if in.Status != nil {
		pStatus = *in.Status
	}
	pAISystemContext := ""
	if in.AISystemContext != nil {
		pAISystemContext = *in.AISystemContext
	}

	p := &Product{
		ID:              in.ID,
		Name:            in.Name,
		Handle:          in.Handle,
		Description:     "",
		Type:            pType,
		Status:          pStatus,
		Details:         pDetails,
		SEO:             pSEO,
		Metadata:        in.Metadata,
		AISystemContext: pAISystemContext,
	}
	if in.Description != nil {
		p.Description = *in.Description
	}

	for _, opt := range in.Options {
		p.Options = append(p.Options, ProductOption{
			Name:   opt.Name,
			Values: opt.Values,
		})
	}

	for _, v := range in.Variants {
		vDetails := ProductDetails{}
		if v.Details != nil {
			vDetails = ProductDetails{
				SKU:                      v.Details.SKU,
				Barcode:                  v.Details.Barcode,
				HSNCode:                  v.Details.HSNCode,
				Weight:                   v.Details.Weight,
				Length:                   v.Details.Length,
				Width:                    v.Details.Width,
				Height:                   v.Details.Height,
				FileUrl:                  v.Details.FileUrl,
				MaxDownloads:             v.Details.MaxDownloads,
				DownloadExpirationHours: v.Details.DownloadExpirationHours,
			}
		}

		var vOpts []VariantOption
		for _, o := range v.Options {
			vOpts = append(vOpts, VariantOption{
				Name:  o.Name,
				Value: o.Value,
			})
		}

		p.Variants = append(p.Variants, ProductVariant{
			Title:          v.Title,
			Price:          v.Price,
			CompareAtPrice: v.CompareAtPrice,
			Details:        vDetails,
			Options:        vOpts,
			Metadata:       v.Metadata,
		})
	}

	for _, img := range in.Images {
		alt := ""
		if img.AltText != nil {
			alt = *img.AltText
		}
		sort := 0
		if img.SortOrder != nil {
			sort = *img.SortOrder
		}
		p.Images = append(p.Images, ProductImage{
			VariantID: img.VariantID,
			URL:       img.URL,
			AltText:   alt,
			SortOrder: sort,
		})
	}

	if err := m.repo.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to save product: %w", err)
	}

	return map[string]any{"product": p}, nil
}

// UpdateProductDetails updates an existing product's information.
func (m *Module) UpdateProductDetails(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	workflowInput, ok := data["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow input")
	}

	productID, _ := workflowInput["id"].(string)
	p, err := m.repo.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	ba, err := json.Marshal(workflowInput)
	if err != nil {
		return nil, err
	}
	var in UpdateProductInput
	if err := json.Unmarshal(ba, &in); err != nil {
		return nil, err
	}

	if in.Name != nil {
		p.Name = *in.Name
	}
	if in.Handle != nil {
		p.Handle = *in.Handle
	}
	if in.Description != nil {
		p.Description = *in.Description
	}
	if in.Type != nil {
		p.Type = *in.Type
	}
	if in.Status != nil {
		p.Status = *in.Status
	}
	if in.AISystemContext != nil {
		p.AISystemContext = *in.AISystemContext
	}
	if in.Metadata != nil {
		p.Metadata = in.Metadata
	}

	if in.Details != nil {
		if in.Details.SKU != nil {
			p.Details.SKU = in.Details.SKU
		}
		if in.Details.Barcode != nil {
			p.Details.Barcode = in.Details.Barcode
		}
		if in.Details.HSNCode != nil {
			p.Details.HSNCode = in.Details.HSNCode
		}
		if in.Details.Weight != nil {
			p.Details.Weight = in.Details.Weight
		}
		if in.Details.Length != nil {
			p.Details.Length = in.Details.Length
		}
		if in.Details.Width != nil {
			p.Details.Width = in.Details.Width
		}
		if in.Details.Height != nil {
			p.Details.Height = in.Details.Height
		}
		if in.Details.FileUrl != nil {
			p.Details.FileUrl = in.Details.FileUrl
		}
		if in.Details.MaxDownloads != nil {
			p.Details.MaxDownloads = in.Details.MaxDownloads
		}
		if in.Details.DownloadExpirationHours != nil {
			p.Details.DownloadExpirationHours = in.Details.DownloadExpirationHours
		}
	}

	if in.SEO != nil {
		if in.SEO.MetaTitle != nil {
			p.SEO.MetaTitle = *in.SEO.MetaTitle
		}
		if in.SEO.MetaDescription != nil {
			p.SEO.MetaDescription = *in.SEO.MetaDescription
		}
		if in.SEO.MetaKeywords != nil {
			p.SEO.MetaKeywords = *in.SEO.MetaKeywords
		}
		if in.SEO.MetaImage != nil {
			p.SEO.MetaImage = *in.SEO.MetaImage
		}
	}

	if err := m.repo.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	return map[string]any{"product": p}, nil
}

// ValidateProductStep wraps ValidateProduct to mdk.StepHandler.
func (m *Module) ValidateProductStep(sCtx mdk.StepContext) mdk.StepResult {
	var inp any = sCtx.Input
	if wfInput, ok := sCtx.Input["input"]; ok {
		inp = wfInput
	}
	res, err := m.ValidateProduct(sCtx.Ctx, map[string]any{
		"input": inp,
	})
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}

// PersistProductStep wraps PersistProduct to mdk.StepHandler.
func (m *Module) PersistProductStep(sCtx mdk.StepContext) mdk.StepResult {
	res, err := m.PersistProduct(sCtx.Ctx, sCtx.Input)
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}

// UpdateProductDetailsStep wraps UpdateProductDetails to mdk.StepHandler.
func (m *Module) UpdateProductDetailsStep(sCtx mdk.StepContext) mdk.StepResult {
	var inp any = sCtx.Input
	if wfInput, ok := sCtx.Input["input"]; ok {
		inp = wfInput
	}
	res, err := m.UpdateProductDetails(sCtx.Ctx, map[string]any{
		"input": inp,
	})
	if err != nil {
		return mdk.StepResult{Err: err}
	}
	resMap, _ := res.(map[string]any)
	return mdk.StepResult{Output: resMap}
}
