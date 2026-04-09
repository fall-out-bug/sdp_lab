# Discovery Hypothesis

**Raw idea:** память агентов по проекту и глобально: дистилляция из диалогов, поиск похожих кейсов, сжатие и гидрация контекста в AI workspace для консультантов

## Test Card (Strategyzer)

**We believe** management consultants at boutique strategy firms need to automatically extract and store key insights from project dialogues because they cannot afford to manually review hundreds of hours of client conversations while maintaining billable utilization targets

**To verify this**, we will 5-question customer interview script with 10 target consultants, asking them to walk through their current process for capturing insights from client conversations and what they wish they could recover

**We'll measure** percentage of interviewees who identify insight capture from dialogues as a top-3 pain point

**We are right if** 7 out of 10 interviewees rank insight capture from client conversations as a top-3 friction point within 14 days

## Assumptions (RAT-Ranked)

| Rank | Assumption | Risk | Uncertainty | RAT Score |
|------|-----------|------|-------------|----------|
| 1 | Firms would allow conversation data to be processed by a third-party tool for insight extraction | high | high | 9 |
| 2 | The extracted insights would be valuable enough for consultants to change their current workflow | medium | high | 6 |
| 3 | Consultants would use a searchable knowledge base if it contained high-quality, structured insights from past projects | high | medium | 6 |
| 4 | Consultants would trust an automated system to identify and extract key insights from conversations | medium | medium | 4 |
| 5 | Consultants struggle to find relevant past cases when working on similar client problems | low | medium | 2 |
| 6 | Consultants have access to transcripts or recordings of client conversations that could be processed | low | low | 1 |

**Riskiest assumption (rank 1):** Firms would allow conversation data to be processed by a third-party tool for insight extraction

## Requirements

- User can upload conversation transcripts and receive structured insight summaries within 60 seconds
- User can search across all past project insights using natural language queries
- User can view similar past cases ranked by relevance to current project context
- User can toggle between compressed summaries and expanded conversation details for any insight
