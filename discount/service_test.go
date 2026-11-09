package discount

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePercent(t *testing.T) {
	want := []struct {
		discountPercent int
		err             error
		name            string
	}{
		{100, nil, "100 discount"},
		{0, nil, "0 discount"},
		{44, nil, "correct discount in range"},
		{-1, ErrInvalidPercent, "negative discount"},
		{101, ErrInvalidPercent, "discount greater than 100"},
	}
	for _, tt := range want {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorIs(t, ValidatePercent(tt.discountPercent), tt.err)
		})
	}
}

func TestApply(t *testing.T) {
	want := []struct {
		price           int
		discountPercent int
		priceWDiscount  int
		err             error
		name            string
	}{
		{100, 3, 97, nil, "correct"},
		{1400, -3, 0, ErrInvalidPercent, "negative discount"},
		{1400, 102, 0, ErrInvalidPercent, "discount greater than 100"},
		{-999, 77, 0, ErrNegativePrice, "negative price"},
		{100, 0, 100, nil, "100 discount"},
		{100, 100, 0, nil, "0 discount"},
	}
	for _, tt := range want {
		t.Run(tt.name, func(t *testing.T) {
			gotPrice, gotErr := Apply(tt.price, tt.discountPercent)
			require.ErrorIs(t, gotErr, tt.err)
			assert.Equal(t, tt.priceWDiscount, gotPrice)
		})
	}
}

func TestFinalPrice(t *testing.T) {
	want := []struct {
		prices []Item
		total  int
		err    error
		name   string
	}{
		{
			[]Item{
				{100, 3}, {-234, -3},
				{-11, 12}, {180, 10},
			},
			0,
			ErrNegativePrice,
			"negative price",
		},
		{
			[]Item{
				{100, 3},
				{234, 0},
				{11, 12},
				{180, 10},
			},
			502,
			nil,
			"correct",
		},
		{
			[]Item{
				{134, 33},
				{13, 0},
				{1334, 100},
				{34, -77},
			},
			0,
			ErrInvalidPercent,
			"invalid percent",
		},
		{
			[]Item{},
			0,
			nil,
			"empty item",
		},
	}
	for _, tt := range want {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			original := make([]Item, len(tt.prices))
			copy(original, tt.prices)
			gotTotal, gotErr := FinalPrice(tt.prices)
			require.ErrorIs(t, gotErr, tt.err)
			assert.Equal(t, tt.total, gotTotal)
			assert.Equal(t, original, tt.prices)
		})
	}
}
