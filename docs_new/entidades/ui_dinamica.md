# 🎨 Server-Driven UI (El Titiritero de la Experiencia)

**Responsabilidad principal:** Enviar interfaces en código, no solo datos aburridos. Las aplicaciones (KMP, SwiftUI) no saben cómo lucir por defecto; es **IAM Platform** quien analiza la mente del usuario (sus poderes y permisos) y despliega la fachada visual exacta que ese usuario debe observar.

De adentro hacia afuera: dictamos *qué menús* ves, *qué campos de formulario* rellenas y *dónde das clic*, eliminando la tortura de someter la app a revisión por Apple y Google cada semana.

## 🧬 Anatomía Rápida del Titiritero:
1. **Recurs0 (`Resource`)**: La entidad conceptual del negocio (Ej. "Módulo Pagos").
2. **Plantilla (`Template`)**: El esqueleto de interfaz (Ej. "Formulario de 2 Columnas").
3. **Instancia (`Instance`)**: La fusión ("Módulo Pagos" dibujado sobre "Formulario 2 Columnas" con configuraciones de botones para "Pagar" y "Cancelar").

---

## 🗺️ Menú Camaleónico: Resolución al Vuelo

El primer saludo entre la App Móvil/Web y el Servidor IAM.

```mermaid
sequenceDiagram
    autonumber
    actor App as 📱 App EduGo (Recién abierta)
    participant IAM as 🛡️ IAM Server-Driven UI 
    participant Acceso as 🎭 IAM Evaluador Permisos
    
    App->>IAM: ¿Cómo debe lucir mi Menú Lateral Principal? (Valida JWT)
    activate IAM
    
    IAM->>IAM: Extrae árbol completo de Menús de la DB
    IAM->>Acceso: Valida usuario contra los menús ("¿Dejo ver el Menú de Finanzas a un Profesor?")
    
    Acceso-->>IAM: Poda el árbol (Elimina elementos sin permiso)
    
    IAM-->>App: ✅ Retorna JSON Dinámico (Solo muestra "Clases", "Notas", "Mi Perfil")
    
    Note over App: La App pinta los botones basándose EXACTAMENTE en lo que envió el servidor.
    deactivate IAM
```

## 🏗️ Renderizado Telepático: El "Resolver"

No es solo el menú; también es **cómo** se ven las pantallas por dentro. Así es como la App sabe pintar un formulario gigantesco si tú eres Admin o uno diminuto si eres estudiante.

```mermaid
sequenceDiagram
    autonumber
    actor User as 🙎‍♂️ Director
    participant App as 📱 EduGo (View Controller)
    participant Resolver as 🛡️ IAM "Resolver" (Key)
    participant DB as 🗄️ IAM DB
    
    User->>App: Clic en Menú "Crear Nueva Materia"
    activate App
    
    App->>Resolver: Resuelve UI para Key: `screen_creacion_materia`
    activate Resolver
    
    Resolver->>DB: Busca configuración entrelazada (Recurso + Instancia)
    DB-->>Resolver: Estructura de componentes puros
    
    Note over Resolver: Evalúa componentes vs Roles. Ejemplo: El botón "Eliminar Base de Datos" de la plantilla, solo para SuperAdmins.
    
    Resolver->>Resolver: Compila JSON Final (UI Layout Dinámico)
    Resolver-->>App: ✅ Retorna Blueprint (Columnas, TextBoxes, Botones, Colores)
    
    App->>User: La pantalla se dibuja instantáneamente frente al usuario y reacciona al layout inyectado.
    deactivate Resolver
    deactivate App
```
