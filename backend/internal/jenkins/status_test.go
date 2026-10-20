package jenkins

import "testing"

func TestNormalizeColor(t *testing.T) {
	tests := []struct {
		color string
		want  Status
	}{
		{color: "blue", want: StatusSuccess},
		{color: "blue_anime", want: StatusBuilding},
		{color: "red", want: StatusFailure},
		{color: "red_anime", want: StatusBuilding},
		{color: "yellow", want: StatusUnstable},
		{color: "yellow_anime", want: StatusBuilding},
		{color: "grey", want: StatusNotBuilt},
		{color: "grey_anime", want: StatusBuilding},
		{color: "disabled", want: StatusDisabled},
		{color: "aborted", want: StatusAborted},
		{color: "aborted_anime", want: StatusBuilding},
		{color: "BLUE", want: StatusSuccess},
		{color: " blue ", want: StatusSuccess},
		{color: "", want: StatusUnknown},
		{color: "purple", want: StatusUnknown},
		{color: "purple_anime", want: StatusUnknown},
	}

	for _, test := range tests {
		t.Run(test.color, func(t *testing.T) {
			if got := NormalizeColor(test.color); got != test.want {
				t.Errorf("NormalizeColor(%q) = %q, want %q", test.color, got, test.want)
			}
		})
	}
}

func TestStatusFromResult(t *testing.T) {
	tests := []struct {
		result   string
		building bool
		want     Status
	}{
		{result: "SUCCESS", want: StatusSuccess},
		{result: "FAILURE", want: StatusFailure},
		{result: "UNSTABLE", want: StatusUnstable},
		{result: "ABORTED", want: StatusAborted},
		{result: "NOT_BUILT", want: StatusNotBuilt},
		{result: "", want: StatusUnknown},
		{result: "SUCCESS", building: true, want: StatusBuilding},
	}
	for _, test := range tests {
		if got := StatusFromResult(test.result, test.building); got != test.want {
			t.Errorf("StatusFromResult(%q, %v) = %q, want %q", test.result, test.building, got, test.want)
		}
	}
}
