package pref

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidThemeID_Pink(t *testing.T) {
	require.True(t, IsValidThemeID("pink"))

	theme := GetThemeByID("pink")
	require.NotNil(t, theme)
	require.Equal(t, "light", theme.Type)
	require.True(t, theme.IsExtra)

	require.False(t, IsValidThemeID("not-a-theme"))
}
