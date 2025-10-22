# Blackjack Bot Logic

## Tổng quan

Module blackjack đã được tích hợp với hệ thống bot logic thông minh để thay thế cơ chế đặt cược ngẫu nhiên cũ. Bot logic mới sử dụng các chiến lược đặt cược và chơi bài thông minh dựa trên:

- Phân tích mẫu đặt cược
- Quản lý rủi ro
- Chiến lược đặt cược tiến bộ (progressive betting)
- Phân tích lịch sử đặt cược
- Basic Blackjack Strategy
- Chiến lược Split, Double Down, và Insurance

## Cấu trúc

### 1. BlackjackBotLogic

Class chính quản lý logic đặt cược và chơi bài của bot:

```go
type BlackjackBotLogic struct {
    bettingStrategy BettingStrategy
    riskTolerance   int
    betHistory      []*pb.BlackjackPlayerBet
    currentBalance  int64
    bettingPatterns map[string]int
    actionHistory   []*pb.BlackjackAction
}
```

### 2. BettingStrategy

Định nghĩa chiến lược đặt cược:

```go
type BettingStrategy struct {
    PreferredBetTypes []pb.BlackjackBetCode
    BetAmountStrategy BetAmountStrategy
    RiskLevel        string
}
```

### 3. BetAmountStrategy

Chiến lược về số tiền đặt cược:

```go
type BetAmountStrategy struct {
    BaseBetPercentage      float64  // Tỷ lệ cơ bản so với balance
    MaxBetPercentage       float64  // Tỷ lệ tối đa so với balance
    ProgressiveBetting     bool     // Có sử dụng đặt cược tiến bộ
    MartingaleMultiplier   float64  // Hệ số tăng cược sau thua
}
```

## Cách sử dụng

### 1. Khởi tạo Bot Logic

```go
// Tạo bot logic mới
botLogic := NewBlackjackBotLogic()

// Thiết lập balance
botLogic.SetBalance(10000)

// Thiết lập mức độ rủi ro
botLogic.SetRiskLevel("moderate") // conservative, moderate, aggressive
```

### 2. Tạo quyết định đặt cược

```go
// Tạo bet hoàn chỉnh
botBet := botLogic.GenerateBotBet()

// Hoặc tạo từng phần riêng biệt
betType := botLogic.DecideBettingType()
amount := botLogic.DecideBetAmount()
```

### 3. Tạo quyết định chơi bài

```go
// Quyết định hành động trong game
action := botLogic.DecideGameAction(playerHand, dealerUpCard, legalActions)

// Quyết định có nên split không
shouldSplit := botLogic.ShouldSplit(playerHand, dealerUpCard, legalActions)

// Quyết định có nên double down không
shouldDouble := botLogic.ShouldDoubleDown(playerHand, dealerUpCard, legalActions)

// Quyết định có nên mua insurance không
shouldInsurance := botLogic.ShouldTakeInsurance(playerHand, dealerUpCard)
```

### 4. Tích hợp vào MatchState

```go
// Trong MatchState
type MatchState struct {
    // ... other fields
    BotLogic *BlackjackBotLogic
}

// Khởi tạo
func NewMatchState(label *pb.Match) MatchState {
    m := MatchState{
        // ... other fields
        BotLogic: NewBlackjackBotLogic(),
    }
    return m
}

// Sử dụng trong BotTurn
func (s *MatchState) BotTurn(v *bot.BotPresence) error {
    if s.BotLogic != nil {
        s.BotLogic.SetBalance(s.Label.Bet.MarkUnit * 100)
        botBet := s.BotLogic.GenerateBotBet()
        botBet.UserId = v.GetUserId()
        // ... process bet
    }
    // ... fallback logic
}

// Sử dụng trong BotAction
func (s *MatchState) BotAction(v *bot.BotPresence, legalActions []pb.BlackjackActionCode) error {
    if s.BotLogic != nil {
        action := s.BotLogic.DecideGameAction(playerHand, dealerUpCard, legalActions)
        // ... process action
    }
    // ... fallback logic
}
```

## Chiến lược đặt cược

### 1. Mức độ rủi ro

#### Conservative (Bảo thủ)
- Base bet: 2% balance
- Max bet: 10% balance
- Risk tolerance: 10-30

#### Moderate (Trung bình)
- Base bet: 5% balance
- Max bet: 20% balance
- Risk tolerance: 30-70

#### Aggressive (Mạo hiểm)
- Base bet: 10% balance
- Max bet: 40% balance
- Risk tolerance: 70-100

### 2. Chiến lược đặt cược tiến bộ

Bot có thể sử dụng chiến lược Martingale:
- Tăng cược gấp đôi sau khi thua
- Giảm cược về mức cơ bản sau khi thắng

### 3. Basic Blackjack Strategy

Bot sử dụng chiến lược cơ bản của Blackjack:

#### Hard Totals
- 17-21: Always Stay
- 16: Stay vs 2-6, Hit vs 7-A
- 15: Stay vs 2-6, Hit vs 7-A
- 12: Stay vs 4-6, Hit vs 2-3, 7-A
- 11: Always Double Down
- 10: Double Down vs 2-9
- 9: Double Down vs 3-6
- 8 và dưới: Always Hit

#### Soft Totals (với Ace)
- A,9 (20): Always Stay
- A,8 (19): Always Stay
- A,7 (18): Stay vs 9-A, Hit vs 2-8
- A,6 (17): Double Down vs 3-6, Hit vs 2, 7-A
- A,5 (16): Double Down vs 4-6, Hit vs 2-3, 7-A
- A,4 (15): Double Down vs 4-6, Hit vs 2-3, 7-A
- A,3 (14): Double Down vs 5-6, Hit vs 2-4, 7-A
- A,2 (13): Double Down vs 5-6, Hit vs 2-4, 7-A

#### Splitting
- A,A: Always Split
- 8,8: Always Split
- 9,9: Split except vs 7, 10, A
- 7,7: Split vs 2-7
- 6,6: Split vs 2-6
- 4,4: Split vs 5-6 only
- 3,3: Split vs 2-7
- 2,2: Split vs 2-7
- 10,10: Never Split
- 5,5: Never Split

## Tùy chỉnh

### 1. Thay đổi mức độ rủi ro

```go
botLogic.SetRiskLevel("aggressive")
```

### 2. Tùy chỉnh chiến lược

```go
strategy := BettingStrategy{
    PreferredBetTypes: []pb.BlackjackBetCode{
        pb.BlackjackBetCode_BLACKJACK_BET_NORMAL,
        pb.BlackjackBetCode_BLACKJACK_BET_DOUBLE,
    },
    BetAmountStrategy: BetAmountStrategy{
        BaseBetPercentage:    0.03,
        MaxBetPercentage:     0.15,
        ProgressiveBetting:   false,
    },
    RiskLevel: "conservative",
}

botLogic.SetBettingStrategy(strategy)
```

### 3. Thiết lập balance

```go
botLogic.SetBalance(50000)
```

## Testing

Chạy test để kiểm tra bot logic:

```bash
cd blackjack-module
go test ./entity -v
```

## Lưu ý

1. **Balance Management**: Bot logic cần được cập nhật balance thường xuyên để đưa ra quyết định chính xác

2. **Fallback Logic**: Hệ thống vẫn giữ logic cũ để đảm bảo tương thích

3. **Randomization**: Bot logic vẫn có yếu tố ngẫu nhiên để tránh bị dự đoán

4. **Performance**: Bot logic được thiết kế để hoạt động hiệu quả với số lượng bet lớn

5. **Basic Strategy**: Bot sử dụng chiến lược cơ bản được chứng minh về mặt toán học

## So sánh với Baccarat Module

| Tính năng | Baccarat | Blackjack |
|-----------|----------|-----------|
| Bot Logic | ✅ Có | ✅ Có |
| Chiến lược đặt cược | Đặt cược thông minh | Đặt cược + chơi bài thông minh |
| Phân tích mẫu | Dựa trên betting cells | Dựa trên betting types |
| Risk Management | ✅ Có | ✅ Có |
| Progressive Betting | ✅ Có | ✅ Có |
| Balance Management | ✅ Có | ✅ Có |
| Basic Strategy | Không | ✅ Có (Blackjack) |
| Split/Double Down | Không | ✅ Có |
| Insurance | Không | ✅ Có |

## Tương lai

Có thể mở rộng bot logic với:

1. **Card Counting**: Đếm bài để điều chỉnh chiến lược
2. **Advanced Patterns**: Phân tích mẫu phức tạp hơn
3. **Multi-Table Strategy**: Chiến lược cho nhiều bàn cùng lúc
4. **Player Profiling**: Phân tích hành vi người chơi thật
5. **Machine Learning**: Sử dụng AI để học từ kết quả game
