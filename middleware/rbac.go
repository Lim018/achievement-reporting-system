package middleware

import (
	"go-fiber/utils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status": "error",
				"error":  "Token tidak ditemukan",
			})
		}

		tokenStr := strings.TrimSpace(authHeader[len("Bearer "):])
		claims, err := utils.ValidateToken(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status": "error",
				"error":  "Token tidak valid atau sudah kadaluarsa",
			})
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("username", claims.Username)
		c.Locals("role", claims.Role)
		c.Locals("permissions", claims.Permissions)

		return c.Next()
	}
}

func RequirePermission(requiredPermission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		permissions, ok := c.Locals("permissions").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "error",
				"error":  "Akses ditolak: tidak ada informasi permission",
			})
		}

		hasPermission := false
		for _, perm := range permissions {
			if perm == requiredPermission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "error",
				"error":  "Akses ditolak: permission tidak mencukupi",
			})
		}

		return c.Next()
	}
}

func RequireAnyPermission(requiredPermissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		permissions, ok := c.Locals("permissions").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "error",
				"error":  "Akses ditolak: tidak ada informasi permission",
			})
		}

		hasPermission := false
		for _, reqPerm := range requiredPermissions {
			for _, perm := range permissions {
				if perm == reqPerm {
					hasPermission = true
					break
				}
			}
			if hasPermission {
				break
			}
		}

		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "error",
				"error":  "Akses ditolak: permission tidak mencukupi",
			})
		}

		return c.Next()
	}
}

func RequireAllPermissions(requiredPermissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		permissions, ok := c.Locals("permissions").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "error",
				"error":  "Akses ditolak: tidak ada informasi permission",
			})
		}

		permissionMap := make(map[string]bool)
		for _, perm := range permissions {
			permissionMap[perm] = true
		}

		for _, reqPerm := range requiredPermissions {
			if !permissionMap[reqPerm] {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"status": "error",
					"error":  "Akses ditolak: permission tidak mencukupi",
				})
			}
		}

		return c.Next()
	}
}

func RequireRole(requiredRole string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "error",
				"error":  "Akses ditolak: tidak ada informasi role",
			})
		}

		if role != requiredRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "error",
				"error":  "Akses ditolak: role tidak sesuai",
			})
		}

		return c.Next()
	}
}

func RequireMultiRole(requiredRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "error",
				"error":  "Akses ditolak: tidak ada informasi role",
			})
		}

		hasRole := false
		for _, reqRole := range requiredRoles {
			if role == reqRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status": "error",
				"error":  "Akses ditolak: role tidak sesuai",
			})
		}

		return c.Next()
	}
}