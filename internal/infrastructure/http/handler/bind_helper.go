package handler

import (
	"github.com/gin-gonic/gin"

	ginmw "github.com/EduGoGroup/edugo-shared/middleware/gin"
)

// bindJSON delega al shared BindJSON para validacion con errores detallados por campo.
func bindJSON(c *gin.Context, v any) error {
	return ginmw.BindJSON(c, v)
}
