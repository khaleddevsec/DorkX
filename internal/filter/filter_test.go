package filter

import (
	"testing"
)

func TestDorkPatterns(t *testing.T) {
	tests := []struct {
		name     string
		dork     string
		url      string
		expected bool
	}{
		{
			name:     "Path inurl matching",
			dork:     "inurl:admin/index.php",
			url:      "https://example.com/admin/index.php",
			expected: true,
		},
		{
			name:     "Subpath inurl matching",
			dork:     "inurl:admin/index.php",
			url:      "https://example.com/portal/admin/index.php?ref=1",
			expected: true,
		},
		{
			name:     "Path inurl negative non-matching",
			dork:     "inurl:admin/index.php",
			url:      "https://example.com/user/index.php",
			expected: false,
		},
		{
			name:     "Param separated multi-query matching",
			dork:     "inurl:php?id=",
			url:      "https://target.com/news/page.php?cat=articles&id=105",
			expected: true,
		},
		{
			name:     "WordPress plugin path",
			dork:     "inurl:wp-content/plugins/",
			url:      "https://blog.example.com/wp-content/plugins/contact-form-7/readme.txt",
			expected: true,
		},
		{
			name:     "Filetype SQL with keywords",
			dork:     "filetype:sql intext:dump",
			url:      "https://archive.target.org/backups/database_2023.sql",
			expected: true,
		},
		{
			name:     "Ext env matching",
			dork:     "ext:env",
			url:      "https://api.company.com/.env",
			expected: true,
		},
		{
			name:     "Site constraint with TLD matching",
			dork:     "site:.gov.bd inurl:page.php?id=",
			url:      "https://dept.subdomain.gov.bd/page.php?id=99",
			expected: true,
		},
		{
			name:     "Site constraint mismatch",
			dork:     "site:.gov.bd inurl:page.php?id=",
			url:      "https://example.com/page.php?id=99",
			expected: false,
		},
		{
			name:     "Negative exclusions filtering",
			dork:     "inurl:php?id= -github -stackoverflow -.gov.bd",
			url:      "https://github.com/project/file.php?id=1",
			expected: false,
		},
		{
			name:     "Negative site exclusion",
			dork:     "inurl:login.php -site:facebook.com",
			url:      "https://facebook.com/login.php",
			expected: false,
		},
		{
			name:     "Quoted dork pattern",
			dork:     `inurl:"admin/login.php"`,
			url:      "https://secure.site.com/admin/login.php",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := ExtractRulesFromDork(tt.dork)
			result := IsValidURL(tt.url, rule)
			if result != tt.expected {
				t.Errorf("Dork [%s] on URL [%s]: expected %v, got %v", tt.dork, tt.url, tt.expected, result)
			}
		})
	}
}
