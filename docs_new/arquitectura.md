# 🏛️ Arquitectura Core: `edugo-api-iam-platform`

¡Bienvenido al motor primario de EduGo! Este documento desglosa la genialidad técnica detrás de **EduGo API IAM Platform**, el guardián absoluto de la identidad, el acceso y la configuración de nuestro ecosistema. Construido para exprimir al máximo el rendimiento con **Go** y la velocidad brutal del framework **Gin**, este servicio implementa una arquitectura limpia y altamente escalable.

## 🚀 Visión General

El servicio **IAM Platform (Identity and Access Management)** no es un simple validador de contraseñas. Es la espina dorsal de la seguridad y el orquestador de cómo las aplicaciones cliente (iOS, Android, Web) descubren y renderizan sus interfaces dinámicamente. 

## 🏗️ Estructura en Capas (Clean Architecture)

Nuestro código en `internal/` no está tirado al azar; sigue una separación estricta de responsabilidades para que escalar y refactorizar sea un placer, no una tortura:

1. **🛡️ Domain (`internal/domain`)**:
   - El corazón del negocio. Aquí no hay frameworks, solo Go puro.
   - Contiene los modelos base e interfaces de los repositorios.
   - **Entidades Vitales**: Permisos, Roles, Recursos, Configuración de Pantallas.

2. **⚙️ Application (`internal/application`)**:
   - El cerebro de la operación. Aquí viven los **Casos de Uso** orquestados en **Servicios** (`service/`).
   - Define los **DTOs** (`dto/`) que actúan como contratos blindados para la transferencia de datos.

3. **🔌 Infrastructure (`internal/infrastructure`)**:
   - Donde la magia toca el metal. Implementaciones concretas de las interfaces del dominio.
   - **Persistence (`persistence/postgres`)**: Uso elegante de **GORM** para dominar PostgreSQL (apuntando a Neon en Cloud).
   - **HTTP (`http/`)**: La puerta de entrada. **Handlers** rápidos como un rayo y **Middlewares** estrictos (JWT, CORS, Error Handling).

4. **🧩 Módulos Transversales (Cross-Cutting)**:
   - **Auth (`internal/auth`)**: Fábrica y bóveda de tokens JWT. Maneja el ciclo de vida real de las sesiones (Refresh Tokens, Claims).
   - **Audit (`internal/audit`)**: El Ojo de Sauron. Un sistema asíncrono que registra cada movimiento dentro de la plataforma, dejándolo grabado en PostgreSQL.
   - **Config (`internal/config`)**: El maestro de las variables de entorno, asegurando que la app se comporte distinto en Local, Staging y Prod.
   - **Container (`internal/container`)**: El director de orquesta (Inyección de Dependencias) que ensambla toda la aplicación en milisegundos durante el arranque.

## 🔄 Procesos Técnicos y Flujos de Trabajo

### 1. 🔐 Autenticación (Login & Refresh)
El ritual de entrada:
- Las credenciales bombardean el endpoint `/api/v1/auth/login`.
- El `AuthHandler` intercepta, el `AuthService` verifica.
- ¿Todo correcto? Boom 💥. Se expide un flamante `Access Token` (JWT), un `Refresh Token` para mantener la fiesta viva, y el blueprint del rol y datos del usuario.

### 2. 🚦 Autorización y Middlewares
Aquí nadie pasa sin identificación:
- Toda ruta protegida choca contra el `JWTAuthMiddleware`. El token se destripa y el contexto (`jwt_claims`) se inyecta en la request.
- Luego, el temido middleware `RequirePermission(enum.Permission...)` evalúa si el usuario es digno (roles vs permisos). 
- Paralelamente, el `AuditMiddleware` no pierde detalle y asíncronamente guarda un registro de la acción. ¡Velocidad sin sacrificar trazabilidad!

### 3. 🎨 Server-Driven UI (Gestión de Pantallas Dinámicas)
La verdadera magia negra de la plataforma:
- Este backend no es un simple aburrido proveedor de JSON de datos. ¡Sirve el ADN de la UI!
- El frontend consulta cómo debe dibujar una pantalla (`Screen Config`), y la IAM Platform orquesta componentes, layouts y permisos para que las aplicaciones cambien en tiempo real, ¡sin enviar una sola actualización a las tiendas de apps de Apple o Google!
