package series

/* This package implements a group builder for series of images.
A series is a group of images with the same radical part in their name.
*/

import (
	"context"
	"time"
	"fmt"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
	"golang.org/x/exp/constraints"
)

// Group groups assets by series, based on the radical part of the name.
// the in channel receives assets sorted by radical, then by date taken.
func Group(ctx context.Context, in <-chan *assets.Asset, out chan<- *assets.Asset, gOut chan<- *assets.Group) {
	currentRadical := ""
	currentGroup := []*assets.Asset{}

	for {
		select {
		case <-ctx.Done():
			return
		case a, ok := <-in:
			if !ok {
				if len(currentGroup) > 0 {
					sendGroup(ctx, out, gOut, currentGroup)
				}
				return
			}

			if r := a.Radical; r != currentRadical {
				if len(currentGroup) > 0 {
					sendGroup(ctx, out, gOut, currentGroup)
					currentGroup = []*assets.Asset{}
				}
				currentRadical = r
			}
			currentGroup = append(currentGroup, a)
		}
	}
}

func sendGroup(ctx context.Context, out chan<- *assets.Asset, outg chan<- *assets.Group, as []*assets.Asset) {
	if len(as) < 2 {
		// Not a series
		sendAsset(ctx, out, as)
		return
	}
	grouping := assets.GroupByOther

	gotJPG := false
	gotRAW := false
	gotHEIC := false
	gotMP4 := false
	gotMOV := false

	cover := 0
	// determine if the group is a burst
	for i, a := range as {
		gotMP4 = gotMP4 || a.Ext == ".mp4"
		gotMOV = gotMOV || a.Ext == ".mov"
		gotJPG = gotJPG || a.Ext == ".jpg"
		gotRAW = gotRAW || filetypes.IsRawFile(a.Ext)
		gotHEIC = gotHEIC || a.Ext == ".heic" || a.Ext == ".heif"

        fmt.Println(a.NameInfo.Base)
        fmt.Println(gotMP4, gotMOV, gotJPG, gotRAW, gotHEIC)

        // Check if the group is a burst
		if grouping == assets.GroupByOther {
			switch a.Kind {
			case assets.KindBurst:
				grouping = assets.GroupByBurst
			}
		}
		if a.IsCover {
			cover = i
		}
	}

	// If we have only two assets, we can try to group them as raw/jpg or heic/jpg
	if len(as) == 2 {
		if grouping == assets.GroupByOther {
			if gotJPG && gotRAW && !gotHEIC {
				grouping = assets.GroupByRawJpg
			} else if gotJPG && !gotRAW && gotHEIC {
				grouping = assets.GroupByHeicJpg
			} else if (gotMP4 || gotMOV) && (gotJPG || gotHEIC) {
				grouping = assets.GroupByNone
			}
		}
		if grouping == assets.GroupByNone {
			for _, a := range as {
				select {
				case out <- a:
				case <-ctx.Done():
					return
				}
			}
        }
	}

    // Process time-based grouping for any asset count
    threshold := 1 * time.Second
    var currentGroup []*assets.Asset // Temporary group buffer
    for _, a := range as {
        if len(currentGroup) == 0 {
            currentGroup = append(currentGroup, a) // Start a new group
            continue
        }

        // Compare timestamps with the last asset in the current group
        lastAsset := currentGroup[len(currentGroup)-1]
        timeDifference := abs(lastAsset.CaptureDate.Sub(a.CaptureDate))
        if timeDifference > threshold { // Too far apart, start a new group
            if len(currentGroup) > 0 {
				if len(currentGroup) == 1 {
					sendAsset(ctx, out, currentGroup)
				} else {
					g := assets.NewGroup(grouping, currentGroup...)
					g.CoverIndex = cover
					select {
					case <-ctx.Done():
						return
					case outg <- g:
					}
				}
			}
            currentGroup = []*assets.Asset{a} // Reset group
        } else {
            currentGroup = append(currentGroup, a) // Add to current group
        }
    }

    // Handle the final group
    if len(currentGroup) > 0 {
        
		if len(currentGroup) == 1 {
			sendAsset(ctx, out, currentGroup)
		} else {
			g := assets.NewGroup(grouping, currentGroup...)
        	g.CoverIndex = cover
			select {
			case <-ctx.Done():
				return
			case outg <- g:
			}
		}
    }
}

// sendAsset sends assets of the group as individual assets to the output channel
func sendAsset(ctx context.Context, out chan<- *assets.Asset, assets []*assets.Asset) {
	for _, a := range assets {
		select {
		case out <- a:
		case <-ctx.Done():
			return
		}
	}
}

func abs[T constraints.Integer](x T) T {
	if x < 0 {
		return -x
	}
	return x
}
