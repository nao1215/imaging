package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/nao1215/imaging"
	"github.com/spf13/cobra"
)

func newSharpenCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "sharpen",
		Short: "Sharpening the image",
		Long: `Sharpening the image. 

		The file extension specified in the --output parameter can be different from the input image's extension.`,
		Example: "   gina sharpen --sigma 2.0 input.jpg",
		RunE:    sharpen,
	}

	cmd.Flags().Float64P("sigma", "s", 0.0, "sigma parameter allows to control the strength of the sharpening effect")
	cmd.Flags().StringP("output", "o", "output.jpg", "output filename (supported format: jpg, png, gif, tiff, bmp)")

	return &cmd
}

// sharpen have options for sharpen image.
type sharpener struct {
	sigma  float64
	input  string
	output string
}

// newSharpener returns a new sharpener. It returns an error if the required options are not set.
func newSharpener(cmd *cobra.Command, args []string) (*sharpener, error) {
	s, err := cmd.Flags().GetFloat64("sigma")
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

	return &sharpener{
		sigma:  s,
		input:  args[0],
		output: o,
	}, nil
}

func sharpen(cmd *cobra.Command, args []string) error {
	sharpener, err := newSharpener(cmd, args)
	if err != nil {
		return err
	}
	return sharpener.sharpen()
}

func (s *sharpener) sharpen() error {
	src, err := imaging.Open(s.input)
	if err != nil {
		return err
	}

	dst := imaging.Sharpen(src, s.sigma)
	fmt.Fprintf(os.Stdout, "save image: %s\n", s.output)
	return imaging.Save(dst, s.output)
}
