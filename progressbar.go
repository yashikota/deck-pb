package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/api/slides/v1"
)

const (
	progressBarDescription = "PROGRESS_BAR_ID"
	emuPerPixel            = 9525 // 1px at 96 DPI = 9525 EMU
)

// ApplyProgressBars deletes existing progress bars and creates new ones.
func ApplyProgressBars(ctx context.Context, srv *slides.Service, presentationID string, cfg *ProgressConfig) error {
	pres, err := srv.Presentations.Get(presentationID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get presentation: %w", err)
	}

	pageWidth := pres.PageSize.Width.Magnitude
	pageHeight := pres.PageSize.Height.Magnitude
	barHeight := float64(cfg.Height) * emuPerPixel

	totalSlides := len(pres.Slides)
	if totalSlides == 0 {
		return fmt.Errorf("presentation has no slides")
	}

	endPage := cfg.EndPage
	if endPage == 0 || endPage > totalSlides {
		endPage = totalSlides
	}
	startPage := cfg.StartPage
	if startPage > totalSlides {
		return fmt.Errorf("startPage (%d) exceeds total slides (%d)", startPage, totalSlides)
	}
	if endPage < startPage {
		return fmt.Errorf("endPage (%d) < startPage (%d)", endPage, startPage)
	}

	var translateY float64
	if cfg.Position == "top" {
		translateY = 0
	} else {
		translateY = pageHeight - barHeight
	}

	rgbColor, err := parseHexColor(cfg.Color)
	if err != nil {
		return fmt.Errorf("invalid color: %w", err)
	}

	var reqs []*slides.Request

	// Delete existing progress bars from all slides
	for _, slide := range pres.Slides {
		for _, elem := range slide.PageElements {
			if elem.Description == progressBarDescription {
				reqs = append(reqs, &slides.Request{
					DeleteObject: &slides.DeleteObjectRequest{
						ObjectId: elem.ObjectId,
					},
				})
			}
		}
	}

	// Create new progress bars for slides in range
	slidesInRange := endPage - startPage + 1
	for i := startPage; i <= endPage; i++ {
		slideIdx := i - 1 // 0-based index
		posInRange := i - startPage // 0-based position within range

		var barWidth float64
		if slidesInRange == 1 {
			barWidth = pageWidth
		} else {
			barWidth = pageWidth * float64(posInRange) / float64(slidesInRange-1)
		}

		// Skip first slide in range (width == 0), matching GAS reference
		if barWidth <= 0 {
			continue
		}

		objectID := fmt.Sprintf("pb_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))
		slideObjectID := pres.Slides[slideIdx].ObjectId

		// Create rectangle shape
		reqs = append(reqs, &slides.Request{
			CreateShape: &slides.CreateShapeRequest{
				ObjectId:  objectID,
				ShapeType: "RECTANGLE",
				ElementProperties: &slides.PageElementProperties{
					PageObjectId: slideObjectID,
					Size: &slides.Size{
						Width:  &slides.Dimension{Magnitude: barWidth, Unit: "EMU"},
						Height: &slides.Dimension{Magnitude: barHeight, Unit: "EMU"},
					},
					Transform: &slides.AffineTransform{
						ScaleX:     1.0,
						ScaleY:     1.0,
						TranslateX: 0,
						TranslateY: translateY,
						Unit:       "EMU",
					},
				},
			},
		})

		// Set fill color and remove outline
		reqs = append(reqs, &slides.Request{
			UpdateShapeProperties: &slides.UpdateShapePropertiesRequest{
				ObjectId: objectID,
				ShapeProperties: &slides.ShapeProperties{
					ShapeBackgroundFill: &slides.ShapeBackgroundFill{
						SolidFill: &slides.SolidFill{
							Color: &slides.OpaqueColor{
								RgbColor: rgbColor,
							},
							Alpha: 1.0,
						},
					},
					Outline: &slides.Outline{
						PropertyState: "NOT_RENDERED",
					},
				},
				Fields: "shapeBackgroundFill.solidFill,outline",
			},
		})

		// Set description as identifier for cleanup
		reqs = append(reqs, &slides.Request{
			UpdatePageElementAltText: &slides.UpdatePageElementAltTextRequest{
				ObjectId:    objectID,
				Description: progressBarDescription,
			},
		})
	}

	if len(reqs) == 0 {
		return nil
	}

	_, err = srv.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: reqs,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to batch update: %w", err)
	}

	return nil
}

// DeleteProgressBars removes all progress bar shapes from the presentation.
func DeleteProgressBars(ctx context.Context, srv *slides.Service, presentationID string) error {
	pres, err := srv.Presentations.Get(presentationID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get presentation: %w", err)
	}

	var reqs []*slides.Request
	for _, slide := range pres.Slides {
		for _, elem := range slide.PageElements {
			if elem.Description == progressBarDescription {
				reqs = append(reqs, &slides.Request{
					DeleteObject: &slides.DeleteObjectRequest{
						ObjectId: elem.ObjectId,
					},
				})
			}
		}
	}

	if len(reqs) == 0 {
		fmt.Println("No progress bars found.")
		return nil
	}

	_, err = srv.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: reqs,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete progress bars: %w", err)
	}

	fmt.Printf("Deleted %d progress bar(s).\n", len(reqs))
	return nil
}

// parseHexColor converts a hex color string like "#4285F4" to an RgbColor.
func parseHexColor(hex string) (*slides.RgbColor, error) {
	if len(hex) != 7 || hex[0] != '#' {
		return nil, fmt.Errorf("invalid hex color: %q", hex)
	}
	r, err := strconv.ParseUint(hex[1:3], 16, 8)
	if err != nil {
		return nil, fmt.Errorf("invalid red component: %w", err)
	}
	g, err := strconv.ParseUint(hex[3:5], 16, 8)
	if err != nil {
		return nil, fmt.Errorf("invalid green component: %w", err)
	}
	b, err := strconv.ParseUint(hex[5:7], 16, 8)
	if err != nil {
		return nil, fmt.Errorf("invalid blue component: %w", err)
	}
	return &slides.RgbColor{
		Red:   float64(r) / 255.0,
		Green: float64(g) / 255.0,
		Blue:  float64(b) / 255.0,
	}, nil
}
