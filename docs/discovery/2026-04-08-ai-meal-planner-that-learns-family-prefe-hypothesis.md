# Discovery Hypothesis

**Raw idea:** AI meal planner that learns family preferences

## Test Card (Strategyzer)

**We believe** parents of children with mixed dietary needs (e.g., vegetarian, gluten-free, allergies) need to generate weekly meal plans that automatically accommodate all restrictions while minimizing prep time because manually cross-referencing recipes with individual needs creates 2+ hours of weekly cognitive overhead

**To verify this**, we will 5-question interview script with 15 target parents, asking them to walk through their current meal-planning process and rate pain points

**We'll measure** percentage of interviewees who cite 'managing multiple dietary needs' as their top frustration

**We are right if** 12 out of 15 interviewees rank cross-referencing dietary restrictions as their #1 pain point within 10 days

## Assumptions (RAT-Ranked)

| Rank | Assumption | Risk | Uncertainty | RAT Score |
|------|-----------|------|-------------|----------|
| 1 | Generated plans can balance nutrition, variety, and prep time effectively | high | high | 9 |
| 2 | Reducing decision fatigue is a strong enough motivator to change meal-planning tools | high | medium | 6 |
| 3 | Families will trust algorithmically generated meal plans over their own curated lists | medium | medium | 4 |
| 4 | Recipe databases can be reliably tagged with sufficient dietary metadata | medium | low | 2 |
| 5 | Parents are willing to input family dietary profiles once to save weekly planning time | low | low | 1 |

**Riskiest assumption (rank 1):** Generated plans can balance nutrition, variety, and prep time effectively

## Requirements

- User can create and save profiles for each family member with dietary restrictions and preferences
- System generates a 7-day meal plan that excludes incompatible ingredients for all profiles
- Plan includes estimated prep times and flags days requiring under 30 minutes of active cooking
- User can swap individual meals while maintaining dietary compliance across the week
