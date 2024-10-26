package filenames

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/metadata"
)

func TestGetInfo(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
		info     NameInfo
	}{
		{
			name:     "Normal",
			filename: "PXL_20231026_210642603.dng",
			expected: true,
			info: NameInfo{
				Radical: "PXL_20231026_210642603",
				Base:    "PXL_20231026_210642603.dng",
				IsCover: false,
				Ext:     ".dng",
				Type:    metadata.TypeImage,
				Taken:   time.Date(2023, 10, 26, 21, 6, 42, 0, time.UTC),
			},
		},
		{
			name:     "Nexus BURST cover",
			filename: "00015IMG_00015_BURST20171111030039_COVER.jpg",
			expected: true,
			info: NameInfo{
				Radical: "BURST20171111030039",
				Base:    "00015IMG_00015_BURST20171111030039_COVER.jpg",
				IsCover: true,
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Kind:    KindBurst,
				Index:   15,
				Taken:   time.Date(2017, 11, 11, 3, 0, 39, 0, time.Local),
			},
		},
		{
			name:     "Samsung BURST",
			filename: "20231207_101605_031.jpg",
			expected: true,
			info: NameInfo{
				Radical: "20231207_101605",
				Base:    "20231207_101605_031.jpg",
				IsCover: false,
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Kind:    KindBurst,
				Index:   31,
				Taken:   time.Date(2023, 12, 7, 10, 16, 5, 0, time.Local),
			},
		},
		{
			name:     "Regular",
			filename: "IMG_20171111_030128.jpg",
			expected: false,
			info: NameInfo{
				Radical: "IMG_20171111_030128",
				Base:    "IMG_20171111_030128.jpg",
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Taken:   time.Date(2017, 11, 11, 3, 1, 28, 0, time.Local),
			},
		},
		{
			name:     "InvalidFilename",
			filename: "IMG_1123.jpg",
			expected: false,
			info: NameInfo{
				Base:    "IMG_1123.jpg",
				Radical: "IMG_1123",
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
			},
		},
	}

	ic := InfoCollector{
		TZ: time.Local,
		SM: metadata.DefaultSupportedMedia,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ic.GetInfo(tt.filename)

			if info != tt.info {
				t.Errorf("expected \n%+v,\n  got \n%+v", tt.info, info)
			}
		})
	}
}