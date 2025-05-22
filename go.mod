module scepter

go 1.21.6

require github.com/marsquick/scepter/migration v0.0.0

replace github.com/marsquick/scepter/migration => ./migration

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.20.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/gorm v1.26.1 // indirect
)
