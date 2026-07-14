package cmd

import (
	"reflect"
	"testing"

	"google.golang.org/api/webmasters/v3"
)

func TestParseDimensions(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
		wantErr  bool
	}{
		{
			name:     "empty input returns default query dimension",
			input:    []string{},
			expected: []string{"query"},
			wantErr:  false,
		},
		{
			name:     "nil input returns default query dimension",
			input:    nil,
			expected: []string{"query"},
			wantErr:  false,
		},
		{
			name:     "valid single dimension",
			input:    []string{"page"},
			expected: []string{"page"},
			wantErr:  false,
		},
		{
			name:     "valid multiple dimensions",
			input:    []string{"query", "device", "country"},
			expected: []string{"query", "device", "country"},
			wantErr:  false,
		},
		{
			name:     "case insensitivity and whitespace trimming",
			input:    []string{"  QUERY ", " Device  ", "COUNTRY"},
			expected: []string{"query", "device", "country"},
			wantErr:  false,
		},
		{
			name:     "search_appearance mapping to searchAppearance",
			input:    []string{"search_appearance"},
			expected: []string{"searchAppearance"},
			wantErr:  false,
		},
		{
			name:     "search_appearance mapping to searchAppearance with mixed case",
			input:    []string{"SeArCh_ApPeArAnCe"},
			expected: []string{"searchAppearance"},
			wantErr:  false,
		},
		{
			name:     "unsupported dimension returns error",
			input:    []string{"unsupported_dim"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "mix of valid and invalid returns error",
			input:    []string{"query", "invalid"},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDimensions(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseDimensions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseDimensions() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseFilters(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []*webmasters.ApiDimensionFilter
		wantErr  bool
	}{
		{
			name:     "empty input returns nil filters",
			input:    []string{},
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "nil input returns nil filters",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:  "valid contains filter",
			input: []string{"query contains seo"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "query",
					Operator:   "contains",
					Expression: "seo",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid equals filter",
			input: []string{"device equals MOBILE"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "device",
					Operator:   "equals",
					Expression: "MOBILE",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid == filter",
			input: []string{"device == MOBILE"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "device",
					Operator:   "equals",
					Expression: "MOBILE",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid notcontains filter",
			input: []string{"page notcontains /blog"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "page",
					Operator:   "notContains",
					Expression: "/blog",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid not_contains filter with underscores",
			input: []string{"page not_contains /blog"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "page",
					Operator:   "notContains",
					Expression: "/blog",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid notequals filter",
			input: []string{"device notequals tablet"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "device",
					Operator:   "notEquals",
					Expression: "tablet",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid not_equals filter with underscores",
			input: []string{"device not_equals tablet"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "device",
					Operator:   "notEquals",
					Expression: "tablet",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid != filter",
			input: []string{"device != tablet"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "device",
					Operator:   "notEquals",
					Expression: "tablet",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid includingregex filter",
			input: []string{"query includingregex ^seo"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "query",
					Operator:   "includingRegex",
					Expression: "^seo",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid regex filter",
			input: []string{"query regex ^seo"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "query",
					Operator:   "includingRegex",
					Expression: "^seo",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid ~ filter",
			input: []string{"query ~ ^seo"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "query",
					Operator:   "includingRegex",
					Expression: "^seo",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid excludingregex filter",
			input: []string{"query excludingregex [0-9]"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "query",
					Operator:   "excludingRegex",
					Expression: "[0-9]",
				},
			},
			wantErr: false,
		},
		{
			name:  "valid !~ filter",
			input: []string{"query !~ [0-9]"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "query",
					Operator:   "excludingRegex",
					Expression: "[0-9]",
				},
			},
			wantErr: false,
		},
		{
			name:  "filter with single and double quotes trimmed",
			input: []string{"device == 'MOBILE'", "query contains \"best seo\""},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "device",
					Operator:   "equals",
					Expression: "MOBILE",
				},
				{
					Dimension:  "query",
					Operator:   "contains",
					Expression: "best seo",
				},
			},
			wantErr: false,
		},
		{
			name:  "filter dimension search_appearance mapped to searchAppearance",
			input: []string{"search_appearance equals AMP"},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "searchAppearance",
					Operator:   "equals",
					Expression: "AMP",
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid filter format (too few parts)",
			input:   []string{"query contains"},
			wantErr: true,
		},
		{
			name:    "invalid filter dimension",
			input:   []string{"invalid_dim == value"},
			wantErr: true,
		},
		{
			name:    "invalid filter operator",
			input:   []string{"query invalid_op value"},
			wantErr: true,
		},
		{
			name:  "whitespace trimming within parts",
			input: []string{"  query   contains   seo results  "},
			expected: []*webmasters.ApiDimensionFilter{
				{
					Dimension:  "query",
					Operator:   "contains",
					Expression: "seo results",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFilters(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFilters() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if len(got) != len(tt.expected) {
					t.Fatalf("parseFilters() length = %d, want %d", len(got), len(tt.expected))
				}
				for i := range got {
					if !reflect.DeepEqual(got[i], tt.expected[i]) {
						t.Errorf("parseFilters()[%d] = %+v, want %+v", i, got[i], tt.expected[i])
					}
				}
			}
		})
	}
}
