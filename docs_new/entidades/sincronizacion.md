# ⚡ Motor de Sincronía (El Oxigenador Móvil)

**Responsabilidad principal:** Sincronizar el estado del mundo entre el servidor en la nube y los miles de dispositivos móviles o clientes web distribuidos por todo el ecosistema EduGo. Asegura que los dispositivos puedan sobrevivir y ser útiles en modo "Offline" (sin internet) al recargas sus bases de datos SQLite locales de forma ultra-eficiente.

El proceso es quirúrgico: no descargamos un giga de datos todos los días; descargamos un paquete maestro inicial (Bundle) y luego solo lo que cambió ayer (Deltas).

---

## 📦 Extracción Total: El Paquete Maestro (Bundle)

Ocurre cuando el usuario instala la app por primera vez o borra la caché. Es una descarga pesada inicial con la configuración del universo EduGo.

```mermaid
sequenceDiagram
    autonumber
    actor App as 📱 App Móvil (Instalada)
    participant Sync as ⚡ IAM Sync Engine
    participant DB as 🗄️ IAM DB (Roles, Permisos, Configs)
    
    App->>Sync: `GET /sync/bundle` (Dame mi base local completa)
    activate Sync
    
    Note right of App: "Acabo de instalarme, no sé nada del mundo".
    
    Sync->>DB: Extrae todas las entidades base y diccionarios estáticos (Catálogo Global)
    DB-->>Sync: 15 Tablas serializadas
    
    Sync->>Sync: Comprime (GZIP) y firma el paquete entero
    Sync-->>App: ✅ Retorna Archivo Bundle (Ej: 15MB) + "Último Timestmap: T1"
    
    App->>App: Escribe Bundle en SQLite Local (Modo Offline Activado)
    deactivate Sync
```

## 💉 Inyección Quirúrgica: Delta Sync (Lo que cambió)

Es el pan de cada día. La app despierta, mira el reloj, y pide al servidor que le envíe únicamente los pedacitos de datos que mutaron desde la última vez que hablaron.

```mermaid
sequenceDiagram
    autonumber
    actor App as 📱 App Móvil (Usuario Diurno)
    participant Sync as ⚡ IAM Sync Engine
    participant Audit as 👁️ Tabla de Auditoría / History
    participant DB as 🗄️ Tablas de Datos
    
    Note over App: La App lleva 2 días sin internet.
    App->>Sync: `POST /sync/delta` (Mi último timestamp fue T1)
    activate Sync
    
    Sync->>Audit: ¿Qué se modificó o eliminó DESDE el timestamp T1?
    Audit-->>Sync: "Se agregó el Rol ID: 5 y se borró el Permiso: P2"
    
    Sync->>DB: Extrae datos Frescos (Solo Rol ID: 5)
    
    Sync->>Sync: Empaqueta: { agregados: [Rol:5], eliminados: [P2: Permiso] }
    Sync-->>App: ✅ Retorna Payload Diminuto (Ej: 15KB) + "Nuevo Timestamp: T2"
    
    App->>App: Borra P2 y Guarda R5 en su SQLite Local. (Sincronizada en 1 Segundo)
    deactivate Sync
```
