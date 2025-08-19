-- Seed data for the 8 core MIOSA agents
INSERT INTO agents (id, name, type, description, capabilities, configuration, execution_order, is_active) VALUES
    -- 1. Communication Agent (Fast responses, UI/UX)
    ('a1000000-0000-0000-0000-000000000001', 
     'Communication Agent', 
     'communication',
     'Handles all user interactions, chat responses, and UI/UX communications with fast model (Llama)',
     ARRAY['chat', 'ui_generation', 'quick_responses', 'conversation_flow'],
     '{"model": "llama-3.1-8b-instant", "max_tokens": 2000, "temperature": 0.7}'::jsonb,
     1,
     true),
    
    -- 2. Analysis Agent (Deep business analysis)
    ('a2000000-0000-0000-0000-000000000002',
     'Analysis Agent',
     'analysis', 
     'Performs deep business analysis, market research, and strategic insights using Kimi K2',
     ARRAY['market_analysis', 'competitor_research', 'business_metrics', 'roi_calculation'],
     '{"model": "moonshotai/kimi-k2-instruct", "max_tokens": 8000, "temperature": 0.3}'::jsonb,
     2,
     true),
    
    -- 3. Strategy Agent (Business planning)
    ('a3000000-0000-0000-0000-000000000003',
     'Strategy Agent',
     'strategy',
     'Creates business strategies, roadmaps, and long-term planning',
     ARRAY['strategic_planning', 'roadmap_creation', 'goal_setting', 'milestone_tracking'],
     '{"model": "moonshotai/kimi-k2-instruct", "max_tokens": 6000, "temperature": 0.4}'::jsonb,
     3,
     true),
    
    -- 4. Development Agent (Code generation)
    ('a4000000-0000-0000-0000-000000000004',
     'Development Agent',
     'development',
     'Generates code, creates applications, and handles technical implementation',
     ARRAY['code_generation', 'app_scaffolding', 'api_creation', 'database_design'],
     '{"model": "moonshotai/kimi-k2-instruct", "max_tokens": 10000, "temperature": 0.2}'::jsonb,
     4,
     true),
    
    -- 5. Quality Agent (Testing and validation)
    ('a5000000-0000-0000-0000-000000000005',
     'Quality Agent',
     'quality',
     'Ensures code quality, runs tests, and validates implementations',
     ARRAY['code_review', 'test_generation', 'security_scanning', 'performance_testing'],
     '{"model": "llama-3.1-8b-instant", "max_tokens": 3000, "temperature": 0.1}'::jsonb,
     5,
     true),
    
    -- 6. Deployment Agent (Infrastructure and deployment)
    ('a6000000-0000-0000-0000-000000000006',
     'Deployment Agent',
     'deployment',
     'Manages deployments, infrastructure, and cloud resources',
     ARRAY['ci_cd_setup', 'cloud_deployment', 'docker_configuration', 'monitoring_setup'],
     '{"model": "llama-3.1-8b-instant", "max_tokens": 2500, "temperature": 0.2}'::jsonb,
     6,
     true),
    
    -- 7. Monitoring Agent (Performance and health)
    ('a7000000-0000-0000-0000-000000000007',
     'Monitoring Agent',
     'monitoring',
     'Monitors application health, performance metrics, and system status',
     ARRAY['health_checks', 'performance_monitoring', 'alert_management', 'log_analysis'],
     '{"model": "llama-3.1-8b-instant", "max_tokens": 2000, "temperature": 0.3}'::jsonb,
     7,
     true),
    
    -- 8. Integration Agent (External services)
    ('a8000000-0000-0000-0000-000000000008',
     'Integration Agent',
     'integration',
     'Handles integrations with external services, APIs, and third-party tools',
     ARRAY['api_integration', 'webhook_setup', 'oauth_configuration', 'data_synchronization'],
     '{"model": "llama-3.1-8b-instant", "max_tokens": 3000, "temperature": 0.3}'::jsonb,
     8,
     true)
ON CONFLICT (id) DO NOTHING;