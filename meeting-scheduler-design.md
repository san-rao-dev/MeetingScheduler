# Meeting Scheduler API - Design Document

## System Overview

This document outlines the design of a REST API that helps distributed teams find optimal meeting times based on participant availability. The system allows organizers to create events with multiple potential time slots and lets participants indicate their availability. It then identifies the best possible meeting times.

## Core Domain Entities

### Event
- Unique identifier
- Title
- Description (optional)
- Organizer ID
- Required duration (e.g., 1 hour)
- Status (active, cancelled)
- Created/updated timestamps

### TimeSlot
- Unique identifier
- Event ID
- Start time (with timezone)
- End time (with timezone)
- Created/updated timestamps

### User
- Unique identifier
- Name
- Email
- Created/updated timestamps

### UserAvailability
- User ID
- Event ID
- TimeSlot ID
- Availability status (available, unavailable)
- Created/updated timestamps

## System Architecture

The system will follow a layered architecture:

1. **API Layer**: HTTP handlers, request/response validation, authentication
2. **Service Layer**: Business logic, coordination between different operations
3. **Repository Layer**: Data persistence and retrieval 
4. **Domain Layer**: Core business entities and business rules

### Technology Stack

- **Language**: Go (as specified)
- **Web Framework**: Gin (lightweight and performant)
- **Database**: PostgreSQL (for ACID compliance and relationship support)
- **API Documentation**: OpenAPI 3.0 / Swagger
- **Containerization**: Docker
- **Infrastructure**: Terraform 
- **CI/CD**: GitHub Actions

## REST API Endpoints

### Event Management

```
POST /api/v1/events
GET /api/v1/events
GET /api/v1/events/{eventId}
PUT /api/v1/events/{eventId}
DELETE /api/v1/events/{eventId}
```

### Time Slot Management

```
POST /api/v1/events/{eventId}/timeslots
GET /api/v1/events/{eventId}/timeslots
PUT /api/v1/events/{eventId}/timeslots/{timeslotId}
DELETE /api/v1/events/{eventId}/timeslots/{timeslotId}
```

### User Availability

```
POST /api/v1/events/{eventId}/users/{userId}/availability
GET /api/v1/events/{eventId}/users/{userId}/availability
PUT /api/v1/events/{eventId}/users/{userId}/availability/{timeslotId}
DELETE /api/v1/events/{eventId}/users/{userId}/availability/{timeslotId}
```

### Recommendations

```
GET /api/v1/events/{eventId}/recommendations
```

## Data Models

### Event Creation Request
```json
{
  "title": "Brainstorming Meeting",
  "description": "Quarterly brainstorming session",
  "requiredDuration": 60,
  "organizerId": "user123"
}
```

### TimeSlot Creation Request
```json
{
  "startTime": "2025-01-12T14:00:00-05:00",
  "endTime": "2025-01-12T16:00:00-05:00"
}
```

### User Availability Request
```json
{
  "timeslotId": "timeslot123",
  "status": "available"
}
```

### Recommendation Response
```json
{
  "recommendations": [
    {
      "timeslot": {
        "id": "timeslot123",
        "startTime": "2025-01-12T14:00:00-05:00",
        "endTime": "2025-01-12T16:00:00-05:00"
      },
      "availableUsers": ["user1", "user2", "user3"],
      "unavailableUsers": [],
      "availabilityPercentage": 100
    },
    {
      "timeslot": {
        "id": "timeslot456",
        "startTime": "2025-01-14T18:00:00-05:00",
        "endTime": "2025-01-14T21:00:00-05:00"
      },
      "availableUsers": ["user1", "user3"],
      "unavailableUsers": ["user2"],
      "availabilityPercentage": 66.67
    }
  ]
}
```

## Implementation Approach

### Time Handling

All times will be stored in UTC in the database. API requests and responses will include timezone information to ensure proper display and handling of times. The system will use the Go `time` package for timezone conversions.

### Recommendation Algorithm

1. Retrieve all time slots for the event
2. For each time slot, determine which users are available
3. Score each time slot based on the number of available users
4. Sort time slots by score (highest to lowest)
5. Return sorted list with availability details

### Database Schema Design

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE events (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    organizer_id UUID NOT NULL REFERENCES users(id),
    required_duration INTEGER NOT NULL, -- in minutes
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE time_slots (
    id UUID PRIMARY KEY,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE user_availability (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    time_slot_id UUID NOT NULL REFERENCES time_slots(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, time_slot_id)
);
```

## Scalability Considerations

### Horizontal Scaling

- Stateless API servers for easy scaling with load balancers
- Database connection pooling for efficient resource usage
- Cache layer for frequently accessed data (e.g., Redis)

### Data Partitioning

- Consider partitioning data by event or date ranges as the system grows
- Use database read replicas for scaling read operations

## Testing Strategy

### Unit Tests

- Test each function in isolation with mocked dependencies
- Focus on core business logic in the service layer
- Use table-driven tests for comprehensive test coverage

### Integration Tests

- Test API endpoints with a test database
- Validate request/response formats
- Test error handling and edge cases

### Load Testing

- Simulate multiple users creating events and updating availability
- Measure response times for recommendation algorithm under load

## Deployment Strategy

### Containerization

Docker containers for reproducible environments:

```Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /meeting-scheduler ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /meeting-scheduler .
EXPOSE 8080
CMD ["./meeting-scheduler"]
```

### Kubernetes Deployment

Use Helm charts for Kubernetes deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: meeting-scheduler
spec:
  replicas: 3
  selector:
    matchLabels:
      app: meeting-scheduler
  template:
    metadata:
      labels:
        app: meeting-scheduler
    spec:
      containers:
      - name: meeting-scheduler
        image: meeting-scheduler:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: db-secrets
              key: host
        # Additional environment variables...
```

### Infrastructure as Code

Terraform for provisioning cloud resources:

```hcl
provider "aws" {
  region = "us-west-2"
}

resource "aws_db_instance" "postgres" {
  allocated_storage    = 20
  storage_type         = "gp2"
  engine               = "postgres"
  engine_version       = "13"
  instance_class       = "db.t3.micro"
  name                 = "meeting_scheduler"
  username             = "postgres"
  password             = var.db_password
  parameter_group_name = "default.postgres13"
  skip_final_snapshot  = true
}

resource "aws_ecr_repository" "meeting_scheduler" {
  name = "meeting-scheduler"
}

# Additional resources for a complete deployment...
```

## Monitoring and Observability

- Implement structured logging (JSON format)
- Set up metrics collection (Prometheus)
- Distributed tracing (OpenTelemetry)
- Health check endpoints for monitoring service status

## Security Considerations

- Implement authentication and authorization (JWT-based)
- Input validation for all API endpoints
- Protect against common web vulnerabilities (OWASP Top 10)
- Secure database credentials using environment variables or a secrets manager
- Rate limiting to prevent abuse

## Future Enhancements

- Calendar integration (Google Calendar, Outlook)
- Email notifications for event updates
- Recurring meeting support
- Conflict detection with existing events
- User preferences for preferred meeting times
