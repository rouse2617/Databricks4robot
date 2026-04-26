// Package swagger contains the generated Swagger docs.
// Run `make swagger` to regenerate.
package swagger

import "github.com/swaggo/swag"

func init() {
	swag.Register(swag.Name, &swag.Spec{
		InfoInstanceName: swag.Name,
		SwaggerTemplate:  `{"swagger":"2.0","info":{"title":"Data Platform API","version":"1.0"}}`,
	})
}
