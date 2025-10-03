<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" /></a>

## listmonk-messenger

Lightweight HTTP server to handle webhooks from [listmonk](https://listmonk.app) and forward it to different messengers.

### Supported messengers

- Pinpoint
- Twilio
- AWS SES - Use `listmonk >= v2.2.0`


### Development

- Build binary

```
make build
```

- Change config.toml and tweak messenger config

Run the binary which starts a server on :8082

```
./listmonk-messenger.bin --config config.toml --msgr pinpoint --msgr ses
```

- Setting up webhooks
  ![](/screenshots/listmonk-setting-up-webhook.png)

- Add messenger specific subscriber atrributes in listmonk
  ![](/screenshots/listmonk-add-subsriber-attrib.png)

- Add plain text template
  ![](/screenshots/listmonk-plain-text-template.png)

- Change campaign messenger
  ![](/screenshots/listmonk-change-campaign-mgr.png)
# Server Configuration
PORT=5000

# Security Keys - MUST be changed to strong, unique values in production
JWT_SECRET="YOUR_VERY_SECURE_AND_UNIQUE_JWT_SECRET_KEY_HERE"
ADMIN_KEY="r1177bd-admin-2025-secure-key"

# Gaming & Financial Settings
SIGNUP_BONUS=50
COMMISSION_RATE=0.05  # 0.05 = 5% commission on every bet
MIN_DEPOSIT=10
MIN_WITHDRAW=100
require("dotenv").config();
const express = require("express");
const bodyParser = require("body-parser");
const bcrypt = require("bcrypt");
const jwt = require("jsonwebtoken");
const { v4: uuidv4 } = require("uuid");

const app = express();
app.use(bodyParser.json());

// --- CONFIGURATION from .env ---
const PORT = process.env.PORT || 5000;
const JWT_SECRET = process.env.JWT_SECRET;
const ADMIN_KEY = process.env.ADMIN_KEY;
const SIGNUP_BONUS = Number(process.env.SIGNUP_BONUS || 50);
const COMMISSION_RATE = Number(process.env.COMMISSION_RATE || 0.05);
const MIN_DEPOSIT = Number(process.env.MIN_DEPOSIT || 10);
const MIN_WITHDRAW = Number(process.env.MIN_WITHDRAW || 100);

// --- IN-MEMORY DATA STORES (Replace with Database in Production) ---
const users = [];       // { id, email, passwordHash, balance, createdAt }
const transactions = []; // { id, userId, type, amount, status, ref, createdAt }
const games = [];       // { id, title, embedUrl, ... }

// --- HELPERS (Token Generation & Auth Middleware) ---
function generateToken(user) {
  return jwt.sign({ userId: user.id }, JWT_SECRET, { expiresIn: "7d" });
}

function authMiddleware(req, res, next) {
  const header = req.headers["authorization"];
  if (!header) return res.status(401).json({ error: "Authorization Header প্রয়োজন" });
  const parts = header.split(" ");
  if (parts.length !== 2 || parts[0].toLowerCase() !== "bearer") return res.status(401).json({ error: "Invalid Token Format" });
  const token = parts[1];
  try {
    const payload = jwt.verify(token, JWT_SECRET);
    const user = users.find(u => u.id === payload.userId);
    if (!user) return res.status(401).json({ error: "User খুঁজে পাওয়া যায়নি" });
    req.user = user;
    next();
  } catch (err) {
    return res.status(401).json({ error: "Invalid Token" });
  }
}

function adminAuth(req, res, next) {
  const key = req.headers["x-admin-key"];
  if (!key || key !== ADMIN_KEY) return res.status(403).json({ error: "Admin অ্যাক্সেস অনুমোদিত নয়" });
  next();
}

// --- CORE ROUTES: AUTHENTICATION ---

// POST /api/auth/signup: ইউজার সাইন আপ এবং ৫০ টাকা বোনাস
app.post("/api/auth/signup", async (req, res) => {
  try {
    const { email, password } = req.body;
    if (!email || !password) return res.status(400).json({ error: "ইমেল ও পাসওয়ার্ড প্রয়োজন" });
    const exists = users.find(u => u.email === email.toLowerCase());
    if (exists) return res.status(409).json({ error: "ইমেলটি ইতিমধ্যেই নিবন্ধিত" });

    const hash = await bcrypt.hash(password, 10);
    const user = {
      id: uuidv4(),
      email: email.toLowerCase(),
      passwordHash: hash,
      balance: 0,
      createdAt: new Date()
    };
    users.push(user);

    // *** SIGNUP BONUS LOGIC ***
    if (SIGNUP_BONUS > 0) {
      user.balance += SIGNUP_BONUS;
      transactions.push({
        id: uuidv4(),
        userId: user.id,
        type: "bonus",
        amount: SIGNUP_BONUS,
        status: "success",
        ref: "Signup Bonus",
        createdAt: new Date()
      });
    }

    const token = generateToken(user);
    res.status(201).json({ 
        token, 
        message: `সাইনআপ সফল! আপনি ${SIGNUP_BONUS} টাকা বোনাস পেয়েছেন।`,
        user: { id: user.id, email: user.email, balance: user.balance } 
    });
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: "সার্ভার এরর" });
  }
});

// POST /api/auth/login: ইউজার লগইন
app.post("/api/auth/login", async (req, res) => {
  const { email, password } = req.body;
  if (!email || !password) return res.status(400).json({ error: "ইমেল ও পাসওয়ার্ড প্রয়োজন" });
  const user = users.find(u => u.email === email.toLowerCase());
  if (!user) return res.status(400).json({ error: "ভুল ক্রেডেনশিয়াল" });
  const ok = await bcrypt.compare(password, user.passwordHash);
  if (!ok) return res.status(400).json({ error: "ভুল ক্রেডেনশিয়াল" });
  const token = generateToken(user);
  res.json({ token, user: { id: user.id, email: user.email, balance: user.balance } });
});


// --- CORE ROUTES: WALLET & TRANSACTIONS ---

// GET /api/me: বর্তমান ইউজার তথ্য
app.get("/api/me", authMiddleware, (req, res) => {
    res.json({ id: req.user.id, email: req.user.email, balance: req.user.balance, createdAt: req.user.createdAt });
});

// GET /api/wallet/balance: ব্যালেন্স চেক
app.get("/api/wallet/balance", authMiddleware, (req, res) => {
    res.json({ balance: req.user.balance });
});

// POST /api/wallet/deposit/request: ডিপোজিট রিকোয়েস্ট (বিকাশ/নগদ/রকেট/ব্যাংক)
app.post("/api/wallet/deposit/request", authMiddleware, (req, res) => {
  const { amount, method, refId } = req.body; // method: bkash, nagad, bank
  if (!amount || amount < MIN_DEPOSIT) return res.status(400).json({ error: `ডিপোজিট কমপক্ষে ${MIN_DEPOSIT} টাকা হতে হবে` });
  
  const txn = {
    id: uuidv4(),
    userId: req.user.id,
    type: "deposit",
    amount,
    status: "pending",
    ref: `Method: ${method || 'N/A'}, Ref ID: ${refId || 'N/A'}`,
    createdAt: new Date()
  };
  transactions.push(txn);
  res.json({ message: "ডিপোজিট রিকোয়েস্ট সফল। অ্যাডমিন অনুমোদনের জন্য অপেক্ষা করুন।", txn });
});

// POST /api/wallet/withdraw/request: উত্তোলনের অনুরোধ
app.post("/api/wallet/withdraw/request", authMiddleware, (req, res) => {
  const { amount, destination } = req.body; // destination: bkash number, bank details etc.
  if (!amount || amount < MIN_WITHDRAW) return res.status(400).json({ error: `উত্তোলন কমপক্ষে ${MIN_WITHDRAW} টাকা হতে হবে` });
  if (amount > req.user.balance) return res.status(400).json({ error: "পর্যাপ্ত ব্যালেন্স নেই" });

  // টাকাটি ব্যালেন্স থেকে "ফ্রিজ" করা হচ্ছে
  req.user.balance -= amount; 

  const txn = {
    id: uuidv4(),
    userId: req.user.id,
    type: "withdraw",
    amount,
    status: "pending",
    ref: destination || null,
    createdAt: new Date()
  };
  transactions.push(txn);
  res.json({ message: "উত্তোলনের অনুরোধ সফল। অ্যাডমিন অনুমোদনের জন্য অপেক্ষা করুন।", txn });
});


// --- CORE ROUTES: GAME & COMMISSION LOGIC ---

// POST /api/game/play/:gameId: গেম খেলা, বাজি কাটা, জেতা ও কমিশন রাখা
app.post("/api/game/play/:gameId", authMiddleware, (req, res) => {
    const { gameId } = req.params;
    const { betAmount } = req.body;
    const user = req.user; 

    if (!betAmount || betAmount <= 0) return res.status(400).json({ error: "বৈধ বাজির পরিমাণ প্রয়োজন।" });
    if (user.balance < betAmount) return res.status(400).json({ error: "পর্যাপ্ত ব্যালেন্স নেই।" });

    // ১. বাজি ধরার টাকা কেটে নেওয়া (Bet)
    user.balance -= betAmount; 
    transactions.push({
        id: uuidv4(), userId: user.id, type: "bet", amount: -betAmount, status: "success", ref: `Game:${gameId}`, createdAt: new Date()
    });

    // ২. ***কমিশন কেটে নেওয়া (আপনার উপার্জন)***
    const commissionAmount = betAmount * COMMISSION_RATE;
    const netBetAmount = betAmount - commissionAmount; 
    
    transactions.push({
        id: uuidv4(), userId: user.id, type: "commission_fee", amount: commissionAmount, status: "success", ref: `Game:${gameId}`, createdAt: new Date()
    });

    // ৩. গেম রেজাল্ট সিমুলেশন (উদাহরণ: 30% উইন চান্স, জিতলে 2x রিটার্ন)
    let winAmount = 0;
    let result = "lose";

    if (Math.random() < 0.3) { 
        winAmount = netBetAmount * 2;
        user.balance += winAmount; 
        result = "win";

        transactions.push({
            id: uuidv4(), userId: user.id, type: "win", amount: winAmount, status: "success", ref: `Game:${gameId}`, createdAt: new Date()
        });
    }
    
    // ৪. রেসপন্স
    res.json({ 
        success: true, 
        gameResult: result === "win" ? `অভিনন্দন! আপনি ${winAmount.toFixed(2)} টাকা জিতেছেন।` : "দুঃখিত, আপনি হেরেছেন।",
        currentBalance: user.balance 
    });
});

// --- ADMIN ROUTES (Protected by ADMIN_KEY) ---

// GET /api/admin/deposits/pending: পেন্ডিং ডিপোজিট তালিকা
app.get("/api/admin/deposits/pending", adminAuth, (req, res) => {
  const pending = transactions.filter(t => t.type === "deposit" && t.status === "pending");
  res.json(pending);
});

// POST /api/admin/deposits/approve: ডিপোজিট অনুমোদন
app.post("/api/admin/deposits/approve", adminAuth, (req, res) => {
  const { txnId } = req.body;
  const txn = transactions.find(t => t.id === txnId && t.type === "deposit");
  if (!txn || txn.status !== "pending") return res.status(400).json({ error: "Invalid/Not Pending Transaction" });

  txn.status = "success";
  const user = users.find(u => u.id === txn.userId);
  if (user) {
    user.balance += txn.amount;
    return res.json({ message: "Deposit approved and balance credited", newBalance: user.balance });
  } else {
    // If user not found, log error and potentially refund
    txn.status = "failed"; 
    return res.status(404).json({ error: "User not found for credit" });
  }
});

// GET /api/admin/earnings: আপনার মোট কমিশন উপার্জন
app.get("/api/admin/earnings", adminAuth, (req, res) => {
    const commissionTxns = transactions.filter(t => t.type === "commission_fee" && t.status === "success");
    const totalEarnings = commissionTxns.reduce((sum, txn) => sum + txn.amount, 0);

    res.json({
        totalEarnings: totalEarnings.toFixed(2),
        message: "কমিশন বাবদ মোট উপার্জন"
    });
});

// --- SERVER START ---
app.get("/", (req, res) => {
  res.send("R1177bd Gaming Backend is running.");
});

app.listen(PORT, () => {
  console.log(`সার্ভার চালু: http://localhost:${PORT}`);
  console.log(`JWT SECRET: ${JWT_SECRET ? 'SET' : 'MISSING!'}`);
});
