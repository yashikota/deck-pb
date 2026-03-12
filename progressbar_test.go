package main

import (
	"testing"
)

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		wantR   float64
		wantG   float64
		wantB   float64
		wantErr bool
	}{
		{"black", "#000000", 0.0, 0.0, 0.0, false},
		{"white", "#FFFFFF", 1.0, 1.0, 1.0, false},
		{"google blue", "#4285F4", 0x42 / 255.0, 0x85 / 255.0, 0xF4 / 255.0, false},
		{"red", "#FF0000", 1.0, 0.0, 0.0, false},
		{"lowercase", "#ff0000", 1.0, 0.0, 0.0, false},
		{"missing hash", "FF0000", 0, 0, 0, true},
		{"too short", "#FFF", 0, 0, 0, true},
		{"too long", "#FFFFFFF", 0, 0, 0, true},
		{"invalid chars", "#GGGGGG", 0, 0, 0, true},
		{"empty", "", 0, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rgb, err := parseHexColor(tt.hex)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHexColor(%q) error = %v, wantErr %v", tt.hex, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if rgb.Red != tt.wantR || rgb.Green != tt.wantG || rgb.Blue != tt.wantB {
				t.Errorf("parseHexColor(%q) = {R:%f, G:%f, B:%f}, want {R:%f, G:%f, B:%f}",
					tt.hex, rgb.Red, rgb.Green, rgb.Blue, tt.wantR, tt.wantG, tt.wantB)
			}
		})
	}
}

func TestProgressConfigDefaults(t *testing.T) {
	cfg := &ProgressConfig{}
	cfg.applyDefaults()

	if cfg.Position != "bottom" {
		t.Errorf("default Position = %q, want %q", cfg.Position, "bottom")
	}
	if cfg.Height != 10 {
		t.Errorf("default Height = %d, want %d", cfg.Height, 10)
	}
	if cfg.Color != "#4285F4" {
		t.Errorf("default Color = %q, want %q", cfg.Color, "#4285F4")
	}
	if cfg.StartPage != 1 {
		t.Errorf("default StartPage = %d, want %d", cfg.StartPage, 1)
	}
	if cfg.EndPage != 0 {
		t.Errorf("default EndPage = %d, want %d", cfg.EndPage, 0)
	}
}

func TestProgressConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ProgressConfig
		wantErr bool
	}{
		{"valid defaults", ProgressConfig{Position: "bottom", Height: 10, Color: "#4285F4", StartPage: 1}, false},
		{"valid top", ProgressConfig{Position: "top", Height: 5, Color: "#FF0000", StartPage: 1}, false},
		{"valid with endPage", ProgressConfig{Position: "bottom", Height: 10, Color: "#4285F4", StartPage: 2, EndPage: 5}, false},
		{"invalid position", ProgressConfig{Position: "left", Height: 10, Color: "#4285F4", StartPage: 1}, true},
		{"zero height", ProgressConfig{Position: "bottom", Height: 0, Color: "#4285F4", StartPage: 1}, true},
		{"invalid color", ProgressConfig{Position: "bottom", Height: 10, Color: "red", StartPage: 1}, true},
		{"startPage zero", ProgressConfig{Position: "bottom", Height: 10, Color: "#4285F4", StartPage: 0}, true},
		{"endPage < startPage", ProgressConfig{Position: "bottom", Height: 10, Color: "#4285F4", StartPage: 5, EndPage: 3}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBarWidthCalculation(t *testing.T) {
	pageWidth := 9144000.0 // standard 10-inch width in EMU

	tests := []struct {
		name         string
		slidesInRange int
		posInRange   int
		wantWidth    float64
	}{
		{"single slide", 1, 0, pageWidth},
		{"first of two", 2, 0, 0},
		{"last of two", 2, 1, pageWidth},
		{"middle of three", 3, 1, pageWidth / 2},
		{"first of ten", 10, 0, 0},
		{"last of ten", 10, 9, pageWidth},
		{"fifth of ten", 10, 4, pageWidth * 4 / 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var barWidth float64
			if tt.slidesInRange == 1 {
				barWidth = pageWidth
			} else {
				barWidth = pageWidth * float64(tt.posInRange) / float64(tt.slidesInRange-1)
			}
			if barWidth != tt.wantWidth {
				t.Errorf("barWidth = %f, want %f", barWidth, tt.wantWidth)
			}
		})
	}
}
