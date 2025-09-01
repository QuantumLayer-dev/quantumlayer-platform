# 🔌 QuantumLayer V2 - API Architecture Documentation

## Table of Contents
1. [API Strategy](#api-strategy)
2. [GraphQL Architecture](#graphql-architecture)
3. [REST API Design](#rest-api-design)
4. [gRPC Internal APIs](#grpc-internal-apis)
5. [API Gateway](#api-gateway)
6. [Authentication & Authorization](#authentication--authorization)
7. [Rate Limiting & Quotas](#rate-limiting--quotas)
8. [API Versioning](#api-versioning)
9. [WebSocket & Real-time](#websocket--real-time)
10. [API Documentation](#api-documentation)

---

## API Strategy

### Multi-Protocol Approach
```yaml
API Protocols:
  GraphQL:
    purpose: Primary external API
    use_cases: [web_app, mobile_app, complex_queries]
    percentage: 70%
    
  REST:
    purpose: Compatibility and webhooks
    use_cases: [third_party_integrations, simple_CRUD]
    percentage: 20%
    
  gRPC:
    purpose: Internal service communication
    use_cases: [microservice_communication, high_performance]
    percentage: 10%
    
  WebSocket:
    purpose: Real-time updates
    use_cases: [live_generation, notifications, collaboration]
```

---

## GraphQL Architecture

### Schema Design
```graphql
# schema.graphql
type Query {
  # User Queries
  me: User!
  user(id: ID!): User
  users(filter: UserFilter, page: Pagination): UserConnection!
  
  # Code Generation Queries
  generation(id: ID!): Generation
  generations(filter: GenerationFilter, page: Pagination): GenerationConnection!
  generationStatus(id: ID!): GenerationStatus!
  
  # Project Queries
  project(id: ID!): Project
  projects(filter: ProjectFilter, page: Pagination): ProjectConnection!
  
  # Analytics Queries
  analytics(timeRange: TimeRange!): Analytics!
  usage(tenantId: ID!): UsageMetrics!
}

type Mutation {
  # Authentication
  signIn(input: SignInInput!): AuthPayload!
  signUp(input: SignUpInput!): AuthPayload!
  refreshToken(token: String!): AuthPayload!
  
  # Code Generation
  generateCode(input: GenerateCodeInput!): GenerationPayload!
  regenerate(id: ID!, modifications: ModificationInput): GenerationPayload!
  cancelGeneration(id: ID!): Boolean!
  
  # Project Management
  createProject(input: CreateProjectInput!): Project!
  updateProject(id: ID!, input: UpdateProjectInput!): Project!
  deleteProject(id: ID!): Boolean!
  
  # Deployment
  deploy(input: DeployInput!): DeploymentPayload!
  rollback(deploymentId: ID!): Boolean!
}

type Subscription {
  # Real-time Generation Updates
  generationProgress(id: ID!): GenerationProgress!
  
  # Deployment Status
  deploymentStatus(id: ID!): DeploymentStatus!
  
  # System Notifications
  notifications(userId: ID!): Notification!
  
  # Collaboration
  projectUpdates(projectId: ID!): ProjectUpdate!
}

# Core Types
type Generation {
  id: ID!
  status: GenerationStatus!
  prompt: String!
  code: String
  language: String!
  framework: String
  quality: QualityMetrics
  cost: Cost
  duration: Int
  createdAt: DateTime!
  user: User!
  project: Project
}

type GenerationStatus {
  phase: Phase!
  progress: Float!
  currentStep: String
  estimatedCompletion: DateTime
  agents: [AgentStatus!]!
}

enum Phase {
  PARSING
  PLANNING
  GENERATING
  VALIDATING
  PACKAGING
  COMPLETE
  FAILED
}

type AgentStatus {
  type: AgentType!
  status: String!
  progress: Float!
  output: String
}

input GenerateCodeInput {
  prompt: String!
  language: String
  framework: String
  options: GenerationOptions
  projectId: ID
}

input GenerationOptions {
  providers: [String!]
  maxTokens: Int
  temperature: Float
  quality: QualityLevel
  speed: SpeedPriority
}
```

### GraphQL Resolvers Architecture
```typescript
// resolvers/generation.resolver.ts
export class GenerationResolver {
  constructor(
    private generationService: GenerationService,
    private authService: AuthService,
    private pubsub: PubSubEngine
  ) {}
  
  @Query()
  @UseGuards(AuthGuard)
  @RateLimit({ points: 100, duration: 60 })
  async generation(
    @Args('id') id: string,
    @Context() ctx: GraphQLContext
  ): Promise<Generation> {
    // Check permissions
    await this.authService.checkAccess(ctx.user, 'generation', id)
    
    // Get from cache first
    const cached = await this.cache.get(`generation:${id}`)
    if (cached) return cached
    
    // Fetch from service
    const generation = await this.generationService.findById(id)
    
    // Cache for future requests
    await this.cache.set(`generation:${id}`, generation, 300)
    
    return generation
  }
  
  @Mutation()
  @UseGuards(AuthGuard, TenantGuard)
  @Transaction()
  async generateCode(
    @Args('input') input: GenerateCodeInput,
    @Context() ctx: GraphQLContext
  ): Promise<GenerationPayload> {
    // Validate input
    await this.validateGenerationInput(input)
    
    // Check quotas
    await this.checkQuotas(ctx.tenant)
    
    // Start generation workflow
    const generation = await this.generationService.generate({
      ...input,
      userId: ctx.user.id,
      tenantId: ctx.tenant.id
    })
    
    // Publish for subscriptions
    await this.pubsub.publish('GENERATION_STARTED', {
      generationProgress: generation
    })
    
    // Track for billing
    await this.billingService.trackUsage({
      tenantId: ctx.tenant.id,
      type: 'generation',
      metadata: generation
    })
    
    return {
      generation,
      estimatedTime: this.estimateTime(input)
    }
  }
  
  @Subscription()
  @UseGuards(AuthGuard)
  async generationProgress(
    @Args('id') id: string,
    @Context() ctx: GraphQLContext
  ): AsyncIterator<GenerationProgress> {
    // Verify access
    await this.authService.checkAccess(ctx.user, 'generation', id)
    
    // Create filtered subscription
    return this.pubsub.asyncIterator(
      `GENERATION_PROGRESS.${id}`
    )
  }
}
```

### DataLoader Pattern
```typescript
// dataloaders/user.loader.ts
export class UserDataLoader {
  private loader: DataLoader<string, User>
  
  constructor(private userService: UserService) {
    this.loader = new DataLoader(async (ids: string[]) => {
      const users = await this.userService.findByIds(ids)
      const userMap = new Map(users.map(u => [u.id, u]))
      return ids.map(id => userMap.get(id))
    })
  }
  
  async load(id: string): Promise<User> {
    return this.loader.load(id)
  }
  
  async loadMany(ids: string[]): Promise<User[]> {
    return this.loader.loadMany(ids)
  }
}
```

---

## REST API Design

### RESTful Endpoints
```yaml
# REST API Structure
/api/v1:
  /auth:
    POST /signin: Sign in user
    POST /signup: Create account
    POST /refresh: Refresh token
    POST /signout: Sign out
    
  /users:
    GET /: List users (admin)
    GET /{id}: Get user
    PUT /{id}: Update user
    DELETE /{id}: Delete user
    GET /me: Current user
    
  /generations:
    GET /: List generations
    POST /: Create generation
    GET /{id}: Get generation
    DELETE /{id}: Cancel generation
    GET /{id}/status: Generation status
    GET /{id}/download: Download code
    
  /projects:
    GET /: List projects
    POST /: Create project
    GET /{id}: Get project
    PUT /{id}: Update project
    DELETE /{id}: Delete project
    POST /{id}/deploy: Deploy project
    
  /webhooks:
    POST /github: GitHub webhook
    POST /stripe: Stripe webhook
    POST /slack: Slack webhook
```

### REST Controller Example
```typescript
// controllers/generation.controller.ts
@Controller('api/v1/generations')
@UseInterceptors(LoggingInterceptor, CacheInterceptor)
export class GenerationController {
  constructor(
    private generationService: GenerationService,
    private validationService: ValidationService
  ) {}
  
  @Get()
  @UseGuards(JwtAuthGuard)
  @ApiOperation({ summary: 'List generations' })
  @ApiResponse({ status: 200, type: [Generation] })
  @Paginate()
  @Cache({ ttl: 60 })
  async listGenerations(
    @Query() query: ListGenerationsDto,
    @Request() req: AuthenticatedRequest
  ): Promise<PaginatedResponse<Generation>> {
    return this.generationService.findAll({
      ...query,
      userId: req.user.id,
      tenantId: req.tenant.id
    })
  }
  
  @Post()
  @UseGuards(JwtAuthGuard, RateLimitGuard)
  @ApiOperation({ summary: 'Create generation' })
  @ApiResponse({ status: 201, type: Generation })
  @ApiResponse({ status: 429, description: 'Rate limit exceeded' })
  @Validate()
  async createGeneration(
    @Body() dto: CreateGenerationDto,
    @Request() req: AuthenticatedRequest
  ): Promise<Generation> {
    // Validate request
    await this.validationService.validateGeneration(dto)
    
    // Check quotas
    await this.checkQuota(req.tenant)
    
    // Create generation
    const generation = await this.generationService.create({
      ...dto,
      userId: req.user.id,
      tenantId: req.tenant.id
    })
    
    // Return with HATEOAS links
    return {
      ...generation,
      _links: {
        self: `/api/v1/generations/${generation.id}`,
        status: `/api/v1/generations/${generation.id}/status`,
        download: `/api/v1/generations/${generation.id}/download`,
        cancel: `/api/v1/generations/${generation.id}/cancel`
      }
    }
  }
  
  @Get(':id')
  @UseGuards(JwtAuthGuard)
  @ApiOperation({ summary: 'Get generation by ID' })
  @ApiParam({ name: 'id', type: 'string' })
  @ApiResponse({ status: 200, type: Generation })
  @ApiResponse({ status: 404, description: 'Generation not found' })
  async getGeneration(
    @Param('id') id: string,
    @Request() req: AuthenticatedRequest
  ): Promise<Generation> {
    const generation = await this.generationService.findById(id)
    
    if (!generation) {
      throw new NotFoundException('Generation not found')
    }
    
    // Check access
    if (generation.userId !== req.user.id) {
      throw new ForbiddenException('Access denied')
    }
    
    return generation
  }
  
  @Delete(':id')
  @UseGuards(JwtAuthGuard)
  @HttpCode(204)
  @ApiOperation({ summary: 'Cancel generation' })
  async cancelGeneration(
    @Param('id') id: string,
    @Request() req: AuthenticatedRequest
  ): Promise<void> {
    await this.generationService.cancel(id, req.user.id)
  }
}
```

---

## gRPC Internal APIs

### Protocol Buffer Definitions
```protobuf
// proto/internal.proto
syntax = "proto3";

package quantumlayer.internal.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";

// Generation Service
service GenerationService {
  rpc Generate(GenerateRequest) returns (stream GenerateResponse);
  rpc GetStatus(StatusRequest) returns (StatusResponse);
  rpc Cancel(CancelRequest) returns (CancelResponse);
  rpc Validate(ValidateRequest) returns (ValidateResponse);
}

message GenerateRequest {
  string request_id = 1;
  string tenant_id = 2;
  string user_id = 3;
  string prompt = 4;
  GenerationOptions options = 5;
  map<string, string> metadata = 6;
}

message GenerationOptions {
  string language = 1;
  string framework = 2;
  repeated string providers = 3;
  int32 max_tokens = 4;
  float temperature = 5;
  QualityLevel quality = 6;
}

enum QualityLevel {
  DRAFT = 0;
  STANDARD = 1;
  HIGH = 2;
  PRODUCTION = 3;
}

message GenerateResponse {
  string request_id = 1;
  oneof update {
    Progress progress = 2;
    Result result = 3;
    Error error = 4;
  }
}

message Progress {
  float percentage = 1;
  string phase = 2;
  string message = 3;
  repeated AgentUpdate agents = 4;
}

message AgentUpdate {
  string agent_type = 1;
  string status = 2;
  float progress = 3;
  string output = 4;
}

// Agent Service
service AgentService {
  rpc SpawnAgent(SpawnRequest) returns (SpawnResponse);
  rpc ExecuteTask(TaskRequest) returns (stream TaskResponse);
  rpc TerminateAgent(TerminateRequest) returns (TerminateResponse);
}

// LLM Service
service LLMService {
  rpc Complete(CompletionRequest) returns (CompletionResponse);
  rpc Embed(EmbeddingRequest) returns (EmbeddingResponse);
  rpc SelectProvider(SelectionRequest) returns (SelectionResponse);
}
```

### gRPC Service Implementation
```go
// grpc/generation_service.go
package grpc

import (
    "context"
    pb "quantumlayer/proto/internal/v1"
)

type GenerationServer struct {
    pb.UnimplementedGenerationServiceServer
    generator *generation.Service
    logger    *zap.Logger
    metrics   *prometheus.Registry
}

func (s *GenerationServer) Generate(
    req *pb.GenerateRequest,
    stream pb.GenerationService_GenerateServer,
) error {
    ctx := stream.Context()
    
    // Create generation context
    genCtx := &generation.Context{
        RequestID: req.RequestId,
        TenantID:  req.TenantId,
        UserID:    req.UserId,
        Prompt:    req.Prompt,
        Options:   convertOptions(req.Options),
    }
    
    // Start generation with progress callback
    err := s.generator.GenerateWithProgress(
        ctx, 
        genCtx,
        func(progress *generation.Progress) error {
            // Stream progress updates
            return stream.Send(&pb.GenerateResponse{
                RequestId: req.RequestId,
                Update: &pb.GenerateResponse_Progress{
                    Progress: convertProgress(progress),
                },
            })
        },
    )
    
    if err != nil {
        // Send error
        return stream.Send(&pb.GenerateResponse{
            RequestId: req.RequestId,
            Update: &pb.GenerateResponse_Error{
                Error: &pb.Error{
                    Code:    int32(codes.Internal),
                    Message: err.Error(),
                },
            },
        })
    }
    
    // Send final result
    result := s.generator.GetResult(req.RequestId)
    return stream.Send(&pb.GenerateResponse{
        RequestId: req.RequestId,
        Update: &pb.GenerateResponse_Result{
            Result: convertResult(result),
        },
    })
}

// Interceptor for auth, logging, metrics
func (s *GenerationServer) UnaryInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {
    start := time.Now()
    
    // Extract metadata
    md, _ := metadata.FromIncomingContext(ctx)
    tenantID := md.Get("tenant-id")[0]
    
    // Add to context
    ctx = context.WithValue(ctx, "tenant_id", tenantID)
    
    // Call handler
    resp, err := handler(ctx, req)
    
    // Record metrics
    s.metrics.RecordRPC(info.FullMethod, time.Since(start), err)
    
    return resp, err
}
```

---

## API Gateway

### Gateway Configuration
```yaml
# kong.yml or envoy.yaml
services:
  - name: qlayer-api
    url: http://qlayer-service:8080
    routes:
      - name: graphql
        paths: ["/graphql"]
        methods: ["POST", "GET"]
        
      - name: rest-v1
        paths: ["/api/v1"]
        strip_path: false
        
    plugins:
      - name: jwt
        config:
          secret_is_base64: false
          claims_to_verify: ["exp", "nbf"]
          
      - name: rate-limiting
        config:
          minute: 100
          hour: 1000
          policy: local
          
      - name: cors
        config:
          origins: ["https://app.quantumlayer.com"]
          methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
          headers: ["Accept", "Content-Type", "Authorization"]
          
      - name: request-transformer
        config:
          add:
            headers:
              - "X-Tenant-ID:$(jwt.tenant_id)"
              - "X-User-ID:$(jwt.user_id)"
              
      - name: response-transformer
        config:
          add:
            headers:
              - "X-Request-ID:$(request_id)"
              - "X-RateLimit-Remaining:$(rate_limit_remaining)"
```

### Custom Gateway Middleware
```typescript
// gateway/middleware.ts
export class GatewayMiddleware {
  // Request routing
  async route(request: Request): Promise<Response> {
    const path = request.path
    
    if (path.startsWith('/graphql')) {
      return this.routeToGraphQL(request)
    } else if (path.startsWith('/api/v1')) {
      return this.routeToREST(request)
    } else if (path.startsWith('/ws')) {
      return this.routeToWebSocket(request)
    }
    
    throw new NotFoundError('Route not found')
  }
  
  // Load balancing
  selectUpstream(service: string): string {
    const upstreams = this.getHealthyUpstreams(service)
    
    if (upstreams.length === 0) {
      throw new ServiceUnavailableError('No healthy upstreams')
    }
    
    // Round-robin selection
    return upstreams[this.counter++ % upstreams.length]
  }
  
  // Circuit breaker
  async callWithCircuitBreaker(
    service: string,
    request: Request
  ): Promise<Response> {
    const breaker = this.breakers.get(service)
    
    return breaker.execute(async () => {
      const upstream = this.selectUpstream(service)
      return await this.httpClient.request(upstream, request)
    })
  }
}
```

---

## Authentication & Authorization

### JWT Token Structure
```typescript
interface JWTPayload {
  // Standard claims
  sub: string      // User ID
  iat: number      // Issued at
  exp: number      // Expiration
  nbf: number      // Not before
  jti: string      // JWT ID
  
  // Custom claims
  tenant_id: string
  email: string
  roles: string[]
  permissions: string[]
  plan: 'free' | 'pro' | 'enterprise'
  
  // Session info
  session_id: string
  ip_address: string
  user_agent: string
}

// Token generation
export class TokenService {
  generateTokenPair(user: User): TokenPair {
    const payload: JWTPayload = {
      sub: user.id,
      tenant_id: user.tenantId,
      email: user.email,
      roles: user.roles,
      permissions: user.permissions,
      plan: user.plan,
      iat: Date.now() / 1000,
      exp: Date.now() / 1000 + 3600, // 1 hour
      jti: uuid()
    }
    
    const accessToken = jwt.sign(payload, this.accessSecret)
    const refreshToken = jwt.sign(
      { sub: user.id, jti: uuid() },
      this.refreshSecret,
      { expiresIn: '30d' }
    )
    
    return { accessToken, refreshToken }
  }
}
```

### RBAC Implementation
```typescript
// auth/rbac.ts
export class RBACService {
  private permissions = {
    'generation:create': ['user', 'admin'],
    'generation:read': ['user', 'admin'],
    'generation:delete': ['user', 'admin'],
    'project:create': ['user', 'admin'],
    'project:read': ['user', 'admin'],
    'project:update': ['user', 'admin'],
    'project:delete': ['user', 'admin'],
    'admin:users': ['admin'],
    'admin:billing': ['admin'],
    'admin:system': ['admin']
  }
  
  canAccess(
    user: User,
    resource: string,
    action: string
  ): boolean {
    const permission = `${resource}:${action}`
    const allowedRoles = this.permissions[permission]
    
    if (!allowedRoles) return false
    
    return user.roles.some(role => allowedRoles.includes(role))
  }
  
  // Resource-based access control
  async canAccessResource(
    user: User,
    resource: any,
    action: string
  ): Promise<boolean> {
    // Owner check
    if (resource.userId === user.id) return true
    
    // Team member check
    if (resource.teamId && user.teams.includes(resource.teamId)) {
      return this.checkTeamPermission(user, resource.teamId, action)
    }
    
    // Admin override
    if (user.roles.includes('admin')) return true
    
    return false
  }
}
```

---

## Rate Limiting & Quotas

### Rate Limiting Strategy
```typescript
// ratelimit/limiter.ts
export class RateLimiter {
  private limits = {
    free: {
      requests_per_minute: 10,
      requests_per_hour: 100,
      requests_per_day: 500,
      concurrent_requests: 2
    },
    pro: {
      requests_per_minute: 100,
      requests_per_hour: 1000,
      requests_per_day: 10000,
      concurrent_requests: 10
    },
    enterprise: {
      requests_per_minute: 1000,
      requests_per_hour: 10000,
      requests_per_day: 100000,
      concurrent_requests: 100
    }
  }
  
  async checkLimit(
    tenantId: string,
    userId: string,
    endpoint: string
  ): Promise<RateLimitResult> {
    const key = `rate:${tenantId}:${userId}:${endpoint}`
    const plan = await this.getPlan(tenantId)
    const limits = this.limits[plan]
    
    // Check concurrent requests
    const concurrent = await this.redis.get(`concurrent:${userId}`)
    if (concurrent >= limits.concurrent_requests) {
      return {
        allowed: false,
        reason: 'Concurrent request limit exceeded',
        retryAfter: 1000
      }
    }
    
    // Check rate limits
    const counts = await this.redis.multi()
      .incr(`${key}:minute`)
      .expire(`${key}:minute`, 60)
      .incr(`${key}:hour`)
      .expire(`${key}:hour`, 3600)
      .incr(`${key}:day`)
      .expire(`${key}:day`, 86400)
      .exec()
    
    if (counts[0] > limits.requests_per_minute) {
      return {
        allowed: false,
        reason: 'Minute rate limit exceeded',
        limit: limits.requests_per_minute,
        remaining: 0,
        reset: Date.now() + 60000
      }
    }
    
    return {
      allowed: true,
      limit: limits.requests_per_minute,
      remaining: limits.requests_per_minute - counts[0],
      reset: Date.now() + 60000
    }
  }
}
```

---

## API Versioning

### Versioning Strategy
```typescript
// versioning/strategy.ts
export class APIVersioning {
  // Header-based versioning
  @Get('/users')
  @Version('1')
  getUsersV1() {
    return this.userService.findAllV1()
  }
  
  @Get('/users')
  @Version('2')
  getUsersV2() {
    return this.userService.findAllV2()
  }
  
  // URL-based versioning
  @Controller('api/v1/users')
  class UsersV1Controller {}
  
  @Controller('api/v2/users')
  class UsersV2Controller {}
  
  // GraphQL versioning
  type Query {
    users: [User!]! @deprecated(reason: "Use usersV2")
    usersV2(filter: UserFilterV2): UserConnectionV2!
  }
}
```

---

## WebSocket & Real-time

### WebSocket Implementation
```typescript
// websocket/gateway.ts
@WebSocketGateway({
  cors: true,
  namespace: '/realtime'
})
export class RealtimeGateway {
  @WebSocketServer()
  server: Server
  
  @SubscribeMessage('subscribe:generation')
  async subscribeToGeneration(
    @MessageBody() data: { generationId: string },
    @ConnectedSocket() client: Socket
  ) {
    // Verify access
    const user = await this.authService.verifySocket(client)
    
    // Join room
    client.join(`generation:${data.generationId}`)
    
    // Send current status
    const status = await this.generationService.getStatus(
      data.generationId
    )
    client.emit('generation:status', status)
  }
  
  // Broadcast updates
  async broadcastProgress(
    generationId: string,
    progress: Progress
  ) {
    this.server
      .to(`generation:${generationId}`)
      .emit('generation:progress', progress)
  }
}
```

---

## API Documentation

### OpenAPI/Swagger
```yaml
# openapi.yaml
openapi: 3.0.0
info:
  title: QuantumLayer API
  version: 1.0.0
  description: AI-powered code generation platform

servers:
  - url: https://api.quantumlayer.com/v1
    description: Production
  - url: https://staging-api.quantumlayer.com/v1
    description: Staging

paths:
  /generations:
    post:
      summary: Create code generation
      operationId: createGeneration
      tags: [Generation]
      security:
        - bearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/GenerationRequest'
      responses:
        '201':
          description: Generation created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Generation'
        '429':
          description: Rate limit exceeded
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'

components:
  schemas:
    Generation:
      type: object
      properties:
        id:
          type: string
          format: uuid
        status:
          type: string
          enum: [pending, processing, completed, failed]
        code:
          type: string
        language:
          type: string
        
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

### GraphQL Documentation
```graphql
"""
Code generation input
"""
input GenerateCodeInput {
  """
  Natural language prompt describing what to generate
  """
  prompt: String! @constraint(minLength: 10, maxLength: 5000)
  
  """
  Target programming language
  """
  language: String @constraint(pattern: "^[a-z]+$")
  
  """
  Framework to use (optional)
  """
  framework: String
  
  """
  Generation options
  """
  options: GenerationOptions
}

"""
Generation options for fine-tuning the output
"""
input GenerationOptions {
  """
  LLM providers to use in order of preference
  """
  providers: [String!] @constraint(minItems: 1, maxItems: 5)
  
  """
  Maximum tokens to generate
  """
  maxTokens: Int @constraint(min: 100, max: 10000)
  
  """
  Temperature for randomness (0.0 to 1.0)
  """
  temperature: Float @constraint(min: 0, max: 1)
}
```

---

*API Architecture Version: 1.0*  
*Last Updated: Current Session*  
*Next Review: Monthly*