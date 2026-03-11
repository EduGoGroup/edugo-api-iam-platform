# 🛡️ Identidad y Sesiones (El Guardián)

**Responsabilidad principal:** Cuidar la puerta y mantener vivas las sesiones legítimas. IAM Platform no solo verifica quién es el usuario, sino que gestiona dinámicamente cómo y cuándo el usuario interactúa con los distintos perfiles (roles) que posee dentro del ecosistema EduGo.

## 🔄 Flujo Core: Login y Emisión de Tokens

El ritual de ingreso evalúa la identidad del usuario y emite credenciales temporales efímeras de alta seguridad (Access Token / JWT) junto con llaves para alargar la sesión sin intervención humana (Refresh Token).

```mermaid
sequenceDiagram
    autonumber
    actor User as 🙎‍♂️ Usuario (App/Web)
    participant IAM as 🛡️ IAM Platform (Auth)
    participant DB as 🗄️ PostgreSQL (Neon)

    User->>IAM: Envía Credenciales (Email/Password)
    activate IAM
    IAM->>DB: Busca Usuario y Hash
    activate DB
    DB-->>IAM: Retorna Hash y Roles Base
    deactivate DB
    
    rect rgb(200, 255, 200)
    IAM->>IAM: Verifica Hash (Bcrypt)
    end
    
    alt Credenciales Inválidas
        IAM-->>User: ❌ 401 Unauthorized
    else Credenciales Válidas
        IAM->>IAM: Genera Access Token (JWT efímero)
        IAM->>IAM: Genera Refresh Token (Bóveda Segura)
        IAM->>DB: Almacena Refresh Token vinculado al Dispositivo
        IAM-->>User: ✅ Retorna JWT, Refresh Token y Datos del Rol
    end
    deactivate IAM
```

## 🎭 La Magia del "Avatar" (Cambio de Contexto)

Una de las joyas del negocio en EduGo. Si eres **Director** de secundaria, pero tienes un hijo en la misma escuela (perfil **Padre**), el sistema te permite alternar tu nivel de poder y vistas en caliente, sin cerrar sesión ni introducir contraseñas de nuevo.

```mermaid
sequenceDiagram
    autonumber
    actor User as 🙎‍♂️ Usuario (Multi-Rol)
    participant App as 📱 App Cliente
    participant IAM as 🛡️ IAM Platform
    
    User->>App: Selecciona perfil "Padre"
    activate App
    App->>IAM: Envía JWT actual + Petición Cambio de Contexto (RolID: Padre)
    activate IAM
    
    IAM->>IAM: Verifica JWT actual (Validez)
    IAM->>IAM: Verifica que Usuario sea dueño de Rol "Padre"
    
    alt Usuario NO posee el Rol
        IAM-->>App: ❌ 403 Forbidden
    else Cambio Permitido
        IAM->>IAM: Genera NUEVO Access Token (Nuevos Claims)
        IAM-->>App: ✅ Retorna Nuevo JWT (Poderes Actualizados)
        App->>User: Recarga Interfaz como "Padre" (Nuevos Menús)
    end
    deactivate IAM
    deactivate App
```

## 🔋 Mantenimiento del Pulso (Refresh Token)

Monitoreo silencioso. Cuando el "Boleto de Acceso" caduca, la aplicación cliente negocia un nuevo boleto detrás de escena usando el pase vitalicio.

```mermaid
sequenceDiagram
    actor App as 📱 App Cliente
    participant IAM as 🛡️ IAM Platform
    participant DB as 🗄️ PostgreSQL
    
    Note over App,IAM: El JWT ha expirado (401 devuelto por un microservicio)
    
    App->>IAM: Pide renovación enviando "Refresh Token"
    activate IAM
    IAM->>DB: Valida existencia y estado del Refresh Token
    
    alt Token Revocado o Expirado
        IAM-->>App: ❌ Obliga Re-Login (Cierre Sesión)
    else Token Activo
        IAM->>IAM: Genera NUEVO Access Token
        IAM-->>App: ✅ Retorna Nuevo JWT (Sesión Extendida)
        Note over App,IAM: La App reintenta la petición original fallida automáticamente.
    end
    deactivate IAM
```
