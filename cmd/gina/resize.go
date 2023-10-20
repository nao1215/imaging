package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-spectest/imaging"
	"github.com/spf13/cobra"
)

func newResizeCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "resize",
		Short: "Resize image",
		Long: `Resize the only one image. 

If you specify either the height or width, the aspect ratio will be maintained during resizing.
The file extension specified in the --output parameter can be different from the input image's
extension.`,
		Example: "   gina resize -W 100 -o output.png input.jpg",
		RunE:    resize,
	}

	cmd.Flags().IntP("width", "W", 0, "width of output image")
	cmd.Flags().IntP("height", "H", 0, "height of output image")
	cmd.Flags().StringP("output", "o", "output.jpg", "output filename (supported format: jpg, png, gif, tiff, bmp)")

	return &cmd
}

// resize have options for resize image.
type resizer struct {
	width  int
	height int
	input  string
	output string
}

// newResizer returns a new resizer. It returns an error if the required options are not set.
func newResizer(cmd *cobra.Command, args []string) (*resizer, error) {
	w, err := cmd.Flags().GetInt("width")
	if err != nil {
		return nil, err
	}

	h, err := cmd.Flags().GetInt("height")
	if err != nil {
		return nil, err
	}

	o, err := cmd.Flags().GetString("output")
	if err != nil {
		return nil, err
	}

	if len(args) == 0 {
		return nil, errors.New("no argument: input image file path is required")
	}

	return &resizer{
		width:  w,
		height: h,
		input:  args[0],
		output: o,
	}, nil
}

func resize(cmd *cobra.Command, args []string) error {
	resizer, err := newResizer(cmd, args)
	if err != nil {
		return err
	}
	return resizer.resize()
}

func (r *resizer) resize() error {
	src, err := imaging.Open(r.input)
	if err != nil {
		return err
	}

	dst := imaging.Resize(src, r.width, r.height, imaging.Lanczos)
	fmt.Fprintf(os.Stdout, "save image: %s\n", r.output)
	return imaging.Save(dst, r.output)
}
