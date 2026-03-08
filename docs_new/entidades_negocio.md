# 💼 Entidades de Negocio (Procesos y Responsabilidades)

Bienvenido al mapa de responsabilidades y procesos cardinales de la **EduGo API IAM Platform**. Aquí desgranamos cómo fluye el negocio, de qué se responsabiliza cada componente dentro del ecosistema EduGo y los flujos gráficos de cada operación crítica. Olvídate de los verbos HTTP y los detalles puramente técnicos (esos viven felices en el Swagger).

Puedes explorar el detalle y los **diagramas de secuencia visuales (Mermaid)** para cada subsistema presionando en los siguientes enlaces:

### 1. [🛡️ Identidad y Sesiones (El Guardián)](./entidades/auth.md)
**Responsabilidad:** Cuidar la puerta, emitir comprobantes indescifrables (Tokens) y gestionar con maestría las transiciones de poder en caliente (Cambio de Avatar / Roles). Mantiene viva la sesión de forma transparente (Refresh Tokens).

### 2. [🎭 Roles y Privilegios (El Juez de Acceso)](./entidades/roles_privilegios.md)
**Responsabilidad:** La columna vertebral del control. Evalúa la legalidad de cada petición y orquesta los grandes bloques de poder (Roles) creados al mezclar cientos de moléculas (Permisos). Si el Juez dice "No", la API dice `403`. 

### 3. [🎨 Server-Driven UI (El Titiritero de la Experiencia)](./entidades/ui_dinamica.md)
**Responsabilidad:** Las apps de cliente no son más que maniquíes vacíos. IAM inyecta los planos vitales; dependiendo del estatus del usuario que se conecte, construye los menús en tiempo real (eliminando links prohibidos) y escupe las configuraciones de diseño de plantillas e interfaces en vivo. Cero actualizaciones en tiendas móviles. 

### 4. [⚡ Motor de Sincronía (El Oxigenador Móvil)](./entidades/sincronizacion.md)
**Responsabilidad:** Asegurar que los dispositivos móviles de todo el continente sobrevivan de forma autónoma construyendo bases offline. Dicta cómo enviar paquetes colosales e iniciales ("Bundles"), y actualizaciones microscópicas y quirúrgicas ("Deltas") para quienes tienen mal internet en la selva o la montaña. 

### 5. [👁️ Trazabilidad y Seguridad Total (La Bitácora Negra / Audit)](./entidades/auditoria.md)
**Responsabilidad:** La memoria fotográfica irrefutable de la plataforma. Observa cada click, cada cambio en base de datos, y cada inicio de sesión, guardando el evento de forma asíncrona para que todo evento administrativo posea un forense exacto (la hora, la IP y el autor).
