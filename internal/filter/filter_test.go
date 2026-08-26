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
			name:     "User Case 1: index.php/admin",
			dork:     "inurl:admin/login.php",
			url:      "https://dhhodisha.in/index.php/admin",
			expected: true,
		},
		{
			name:     "User Case 2: admin/index.php",
			dork:     "inurl:admin/login.php",
			url:      "http://carboncalculator.americanpistachios.org/admin/index.php",
			expected: true,
		},
		{
			name:     "User Case 3: secondary/admin/login.php",
			dork:     "inurl:admin/login.php",
			url:      "https://www.bamesschool.com.ng/secondary/admin/login.php",
			expected: true,
		},
		{
			name:     "User Case 4: admin/login.php",
			dork:     "inurl:admin/login.php",
			url:      "https://www.mithras-europe.eu/admin/login.php",
			expected: true,
		},
		{
			name:     "User Case 5: login.php",
			dork:     "inurl:admin/login.php",
			url:      "https://registration.bditec.gov.bd/login.php",
			expected: true,
		},
		{
			name:     "User Case 6: admin/login.php",
			dork:     "inurl:admin/login.php",
			url:      "https://www.hamch.in/admin/login.php",
			expected: true,
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
			name:     "Negative exclusions filtering",
			dork:     "inurl:php?id= -github -stackoverflow -.gov.bd",
			url:      "https://github.com/project/file.php?id=1",
			expected: false,
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
