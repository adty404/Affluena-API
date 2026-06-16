# QA Integration Matrix

This document provides a testing checklist for QA and Frontend engineers to ensure that the Affluena UI adheres to the backend's strict business rules.

## Foundation & Auth
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Register** | Submit new valid user (email & password) | 201 Created. User logged in automatically. | Duplicate email throws 409. |
| **Login** | Valid credentials | 200 OK. Returns access and refresh tokens. | Wrong password throws 401. |
| **Protected Route**| Access without token | 401 Unauthorized. | Missing Bearer header. |
| **Token Refresh**| Refresh using valid refresh token| 200 OK. New token pair generated. | Expired refresh token returns 401. |

## Wallet & Permissions
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Wallet**| Submit valid wallet data | 201 Created. Initial balance is 0. | `type="goal"` is explicitly rejected (400). |
| **Update Wallet**| Edit name or type | 200 OK. | Balance cannot be modified directly via PUT. |
| **Delete Wallet**| Delete empty wallet | 200 OK. Wallet is soft-deleted. | Non-empty wallet deletion may fail depending on constraint logic. |
| **Share Wallet** | Invite another user via email | 200 OK. Invite created. | Self-invite rejected. Non-existent user rejected. |
| **Accept Share** | Member accepts invite | 200 OK. Member can now see and use wallet. | - |
| **Cross-User Auth**| Try accessing another user's wallet| 403 Forbidden or 404 Not Found. | IDs manipulated in URL/Request payload. |

## Transactions
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Income**| Income to Wallet A | Wallet A balance increases by amount. | - |
| **Create Expense**| Expense from Wallet A | Wallet A balance decreases by amount. | - |
| **Transfer** | Transfer A -> B | Wallet A decreases, Wallet B increases. | Requires `to_wallet_id`. Must own both wallets. |
| **Adjustment** | Adjust Wallet A | Calculates difference and creates adjustment transaction. | - |
| **Split Bill** | Split expense | Debt generated automatically. | Split total must equal transaction amount. |
| **Data Isolation**| Submit expense to unowned wallet| 403 Forbidden. | - |

## Budget
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Budget**| Submit budget for Category A | 201 Created. | Duplicate category budget in same month rejected. |
| **Expense vs Budget**| Submit personal expense in Category A| Dashboard/Budget view shows budget utilized. | - |
| **Shared Wallet Budget**| Shared member submits expense in Category A| Owner's personal budget is NOT decremented. | Budgets are isolated to personal categories. |
| **Alerts** | Expense exceeds budget limit | Alert generated in Activity Log. | - |

## Debt
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Payable**| Submit payable debt | 201 Created. | Requires valid categories. |
| **Pay Debt** | Partial payment | Remaining amount decreases. Transaction created. | Payment amount > remaining amount rejected. |
| **Cancel Debt**| Cancel an unpaid debt | Debt marked as soft-cancelled/deleted. | Cancel fails if any payment has already been made. |

## Goals
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Goal**| Submit new goal | 201 Created. Goal wallet generated automatically. | - |
| **Invite Member**| Owner invites member | Member added to goal and goal wallet. | - |
| **Contribute** | Transfer from personal to Goal wallet | Goal progress increases. | Unauthorized transfer to goal wallet rejected. |

## Recurring Transactions
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Rule**| Submit schedule | Rule active. Next run calculated. | - |
| **Manual Run** | Execute rule immediately | Transaction created. Balances updated. | - |
| **Pause Rule** | Pause a running schedule | Rule status becomes paused. | - |

## Reports & Export
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Export CSV** | Valid date range | Returns CSV file stream. | Invalid dates 400. |
| **Export Isolation**| - | Only contains data accessible to authenticated user. | - |
