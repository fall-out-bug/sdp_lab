# Discovery Hypothesis

**Raw idea:** улучшить продуктовый вижн агентов Faust Consulting: коммуникация между агентами, выбор моделей, режимы работы агентов

## Test Card (Strategyzer)

**We believe** AI product teams at consulting firms need to configure agent teams with optimized communication, model selection, and operational modes because they waste 40+ hours per project on manual coordination and suboptimal performance

**To verify this**, we will 5-question interviews with 3 AI product managers at consulting firms about their current agent coordination pain points and willingness to test a configuration template

**We'll measure** willingness to test a prototype

**We are right if** 2 out of 3 interviewees agree to test a simple configuration template within 7 days and identify at least 3 specific coordination pain points

## Assumptions (RAT-Ranked)

| Rank | Assumption | Risk | Uncertainty | RAT Score |
|------|-----------|------|-------------|----------|
| 1 | Product managers can define communication protocols without deep technical expertise | medium | high | 6 |
| 2 | Consulting firms have standardized project types that could use predefined agent team configurations | medium | medium | 4 |
| 3 | System architects value flexible operational modes enough to change their current workflow | medium | medium | 4 |
| 4 | AI product teams struggle with manual agent coordination more than with individual agent capabilities | high | low | 3 |
| 5 | ML engineers prioritize model selection optimization over building custom models from scratch | low | medium | 2 |
| 6 | The market has existing tools that could be integrated rather than building everything from scratch | low | low | 1 |

**Riskiest assumption (rank 1):** Product managers can define communication protocols without deep technical expertise

## Requirements

- Product Manager can define and save communication protocols between specific agent types
- ML Engineer can select from pre-configured model options optimized for specific agent tasks
- System Architect can switch between autonomous, collaborative, and human-in-the-loop modes per project
- Team can save and reuse successful agent team configurations across similar projects
