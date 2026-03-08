# 🎭 Roles y Privilegios (El Juez de Acceso)

**Responsabilidad principal:** Dictaminar qué acciones te están permitidas dentro del sistema EduGo, y rechazar sin piedad aquello para lo que no tienes competencia legal. Funciona como un escudo hipermodularizado frente a los otros microservicios de la cadena (Admin API, Mobile API).

## 🧩 Concepción: Roles vs Permisos

- **Rol:** Es solo un contenedor vacío, una máscara amigable. *Ej: "Coordinador Académico"*.
- **Permiso:** Es el átomo de la seguridad. Representa el derecho a ejecutar un verbo sobre un módulo. *Ej: `puede_crear_usuarios`, `puede_ver_reportes_financieros`*.

IAM Platform rompe la rigidez clásica. Un rol puede mutar: si a mitad de año un colegio decide que los Coordinadores ahora pueden expulsar alumnos, IAM simplemente adhiere el permiso `puede_expulsar_estudiantes` al rol "Coordinador", y automáticamente todos los coordinadores del continente heredan el poder.

## 🚦 Flujo Corto: Evaluación de Permisos Restrictiva

Cuando un usuario intenta hacer algo serio, el Juez de Acceso evalúa la legalidad de la acción de forma casi imperceptible gracias a la inyección de claims.

```mermaid
sequenceDiagram
    autonumber
    actor User as 🙎‍♂️ Profesor
    participant App as 📱 EduGo Admin (Frontend)
    participant OtherAPI as ⚙️ Admin API / Mobile API
    participant IAM as 🛡️ IAM Platform (Middleware)

    User->>App: Clic en botón "Eliminar Grupo Clase"
    activate App
    App->>OtherAPI: Petición `DELETE /grupos/5` (Envía JWT)
    activate OtherAPI
    
    Note over OtherAPI,IAM: La petición debe pasar por el Middleware de IAM importado (`RequirePermission`).
    
    OtherAPI->>IAM: Intercepta: ¿Este JWT tiene el permiso `puede_eliminar_grupo`?
    activate IAM
    
    IAM->>IAM: Extrae Rol inyectado en el Token
    IAM->>IAM: Comprueba matriz de Permisos del Rol
    
    alt No tiene el permiso
        IAM-->>OtherAPI: Bloquear Petición (Abortar)
        OtherAPI-->>App: ❌ 403 Forbidden (No autorizado)
        App-->>User: "No tienes privilegios para esto"
    else Tiene el permiso
        IAM-->>OtherAPI: Permitir Paso
        OtherAPI->>OtherAPI: Ejecuta Lógica de Negocio (Borra Grupo)
        OtherAPI-->>App: ✅ 200 OK (Grupo Borrado)
    end
    
    deactivate IAM
    deactivate OtherAPI
    deactivate App
```

## 🛠️ Flujo Administrativo: Forja Mágica de Roles (Asignación en Lote)

Las secretarías de educación no crean permisos de uno por uno; compran "Combos" (Bulks) que definen drásticamente la estructura de poder de toda la institución.

```mermaid
sequenceDiagram
    actor Admin as 👑 Super Administrador
    participant IAM as 🛡️ IAM Platform (Gestión Roles)
    participant DB as 🗄️ PostgreSQL
    
    Admin->>IAM: Sube nueva estructura del Rol "Director Regional" (150 Permisos)
    activate IAM
    
    IAM->>IAM: Valida Semántica (Todos los permisos deben existir)
    Note right of IAM: Operación Bulk Rebuild
    
    IAM->>DB: Inicia Transacción
    IAM->>DB: Elimina (Vacia) matriz previa de permisos del Rol
    IAM->>DB: Inserta 150 nuevos registros en tabla pivote `role_permissions`
    IAM->>DB: Hace COMMIT
    
    IAM-->>Admin: ✅ Rol Habilitado y Listo para Expansión Masiva
    deactivate IAM
```
