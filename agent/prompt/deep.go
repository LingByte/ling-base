package prompt

import "fmt"

// DeepThinkingPrompt wraps a user question in the "deep thinking" prompt
// template that instructs the model to use a bidirectional steel-man
// argumentation approach before answering.
//
// The prompt asks the model to:
//  1. Restate the real problem the user is trying to solve.
//  2. Give the strongest arguments both for and against the user's current
//     idea (steel-man on both sides).
//  3. Identify the real points of disagreement and the key variables most
//     likely to change the conclusion.
//  4. Ask exactly one critical question, then wait for the answer before
//     giving a final judgment.
//
// Used by the /deep TUI command and the --deep CLI flag.
func DeepThinkingPrompt(question string) string {
	return fmt.Sprintf(`先别急着回答，也别默认我已经把问题想清楚。请先对这个问题做一次详细思考的"双向钢人论证"：

1. 用最完整、最有力的方式，重述我真正想解决的问题。
2. 使用钢人论证法分别给出支持我当前想法、以及反对它的最强论证。
3. 找出双方真正的分歧，以及最可能改变结论的关键变量。
4. 只问我一个最关键的问题。等我回答后，再给出明确判断、理由和下一步行动。

我的问题是：%s`, question)
}
