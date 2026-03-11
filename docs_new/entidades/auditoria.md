# 👁️ Trazabilidad y Seguridad (La Bitácora Negra | Audit)

**Responsabilidad principal:** Ser el Historiador Absoluto, silencioso e incorruptible de EduGo. Cada pulsación en cualquier pantalla regulada, en cualquier parte del mundo, es registrada aquí. 

Su genialidad radica en ser **asíncrono**: vigila y anota sin ralentizar las operaciones críticas del sistema. El usuario ni se entera de que cada "Clic en Guardar" quedó sellado con la IP, Fecha y los datos exactos que alteró.

---

## 📸 El Ojo Observador: Registro de Evento Asíncrono

El usuario hace su trabajo, la API se lo permite (o se lo deniega) y la Bitácora guarda una copia exacta del suceso sin estorbar el flujo TCP/HTTP original.

```mermaid
sequenceDiagram
    autonumber
    actor User as 🙎‍♂️ Profesor (IP: 192.168.1.1)
    participant App as 📱 EduGo App
    participant Hand as 🛡️ API Handler (Negocio)
    participant Mid as 👁️ Audit Middleware (Interceptor)
    participant DB as 🗄️ DB (Tabla Audit)
    
    User->>App: Guarda Notas de Matemáticas
    App->>Mid: Petición `POST /calificaciones`
    activate Mid
    
    Mid->>Mid: Clona Cuerpo de Petición (Payload) y Headers
    Mid->>Hand: Pasa la petición limpia (El negocio no sabe que lo observan)
    activate Hand
    
    Hand-->>Mid: Responde: `201 Objeto Creado (ID: 99)`
    deactivate Hand
    
    Mid-->>App: Retorna respuesta inmediata al Usuario (20ms)
    
    Note over App: "Las notas se guardaron con éxito" (Usuario feliz)
    
    rect rgb(255, 230, 230)
    Note over Mid,DB: Proceso Secundario Asíncrono (Goroutine)
    Mid->>DB: Inserta Fila (Quien: "Profesor", Qué: "Crear Nota", Cuándo: AHORA, Resultado: Éxito 201)
    end
    
    deactivate Mid
```

## 🔍 Consulta Forense: Auditoría Estricta

Ocurre cuando un Director de Escuela quiere saber "Quién rayos borró la nota del alumno de 1er Grado".

```mermaid
sequenceDiagram
    autonumber
    actor Dev as 🕵️‍♂️ IT Admin / Auditor
    participant IAM as 🛡️ IAM Audit Endpoint
    participant DB as 🗄️ Base de Datos Audit
    
    Dev->>IAM: Consultar Eventos para Entidad `CALIFICACION` con ID `99` (Últimas 24 Hrs)
    activate IAM
    
    IAM->>DB: Busca historial completo indexado para `EntityID: 99`
    DB-->>IAM: 3 Registros Encontrados (Creación, 2 Modificaciones)
    
    IAM-->>Dev: ✅ Retorna JSON (Historial)
    
    Note over Dev: "Ah! Fue el usuario ID:44, desde la IP XYZ, a las 3:15 AM".
    deactivate IAM
```
