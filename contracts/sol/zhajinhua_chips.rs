use anchor_lang::prelude::*;
use anchor_lang::system_program;

declare_id!("ZHJChips1111111111111111111111111111111111111");

/// Lamports per SOL constant.
const LAMPORTS_PER_SOL: u64 = 1_000_000_000;

/// Game points awarded per 1 USD worth of SOL.
const POINTS_PER_USD: u64 = 1_000;

#[program]
pub mod zhajinhua_chips {
    use super::*;

    /// Initialize the global game state and PDA vault.
    ///
    /// Must be called once by the admin before any other instruction.
    /// `sol_per_usd` is expressed with 9 decimals of precision
    /// (e.g. 0.005 SOL/USD => 5_000_000).
    pub fn initialize(ctx: Context<Initialize>, sol_per_usd: u64) -> Result<()> {
        require!(sol_per_usd > 0, ChipError::InvalidPrice);

        let game_state = &mut ctx.accounts.game_state;
        game_state.admin = ctx.accounts.admin.key();
        game_state.sol_per_usd = sol_per_usd;
        game_state.total_sol_collected = 0;
        game_state.total_chips_sold = 0;
        game_state.vault_bump = ctx.bumps.vault;
        game_state.state_bump = ctx.bumps.game_state;

        msg!(
            "Zhajinhua chip system initialized. sol_per_usd={}",
            sol_per_usd
        );
        Ok(())
    }

    /// Purchase game chips by sending SOL to the PDA vault.
    ///
    /// The number of chips received is calculated as:
    ///   usd_value  = sol_amount / sol_per_usd
    ///   chips      = usd_value * POINTS_PER_USD
    ///
    /// Both `sol_per_usd` and `sol_amount` use lamport precision (9 decimals),
    /// so the division naturally yields a dimensionless USD value scaled by 1e9;
    /// we then multiply by POINTS_PER_USD and divide out the 1e9 scaling.
    pub fn buy_chips(ctx: Context<BuyChips>, sol_amount: u64) -> Result<()> {
        require!(sol_amount > 0, ChipError::ZeroAmount);

        let game_state = &ctx.accounts.game_state;
        let sol_per_usd = game_state.sol_per_usd;

        // chips = (sol_amount * POINTS_PER_USD) / sol_per_usd
        // Use u128 intermediate to avoid overflow on large purchases.
        let chips: u64 = (sol_amount as u128)
            .checked_mul(POINTS_PER_USD as u128)
            .ok_or(ChipError::MathOverflow)?
            .checked_div(sol_per_usd as u128)
            .ok_or(ChipError::MathOverflow)?
            .try_into()
            .map_err(|_| ChipError::MathOverflow)?;

        require!(chips > 0, ChipError::PurchaseTooSmall);

        // Transfer SOL from buyer to the PDA vault.
        system_program::transfer(
            CpiContext::new(
                ctx.accounts.system_program.to_account_info(),
                system_program::Transfer {
                    from: ctx.accounts.buyer.to_account_info(),
                    to: ctx.accounts.vault.to_account_info(),
                },
            ),
            sol_amount,
        )?;

        // Update user account.
        let user_account = &mut ctx.accounts.user_account;
        if user_account.owner == Pubkey::default() {
            user_account.owner = ctx.accounts.buyer.key();
            user_account.bump = ctx.bumps.user_account;
        }
        user_account.total_chips_purchased = user_account
            .total_chips_purchased
            .checked_add(chips)
            .ok_or(ChipError::MathOverflow)?;
        user_account.total_sol_spent = user_account
            .total_sol_spent
            .checked_add(sol_amount)
            .ok_or(ChipError::MathOverflow)?;
        user_account.last_purchase_slot = Clock::get()?.slot;

        // Update global state.
        let game_state = &mut ctx.accounts.game_state;
        game_state.total_sol_collected = game_state
            .total_sol_collected
            .checked_add(sol_amount)
            .ok_or(ChipError::MathOverflow)?;
        game_state.total_chips_sold = game_state
            .total_chips_sold
            .checked_add(chips)
            .ok_or(ChipError::MathOverflow)?;

        msg!(
            "Purchased {} chips for {} lamports (buyer={})",
            chips,
            sol_amount,
            ctx.accounts.buyer.key()
        );
        Ok(())
    }

    /// Admin withdraws SOL from the PDA vault.
    ///
    /// The vault is a PDA so we sign the transfer with vault seeds.
    pub fn withdraw(ctx: Context<Withdraw>, amount: u64) -> Result<()> {
        require!(amount > 0, ChipError::ZeroAmount);

        let vault = &ctx.accounts.vault;
        let rent_exempt_minimum = Rent::get()?.minimum_balance(0);
        let available = vault
            .lamports()
            .checked_sub(rent_exempt_minimum)
            .ok_or(ChipError::InsufficientVaultBalance)?;
        require!(amount <= available, ChipError::InsufficientVaultBalance);

        // Build signer seeds for the vault PDA.
        let vault_bump = ctx.accounts.game_state.vault_bump;
        let seeds: &[&[&[u8]]] = &[&[b"vault", &[vault_bump]]];

        system_program::transfer(
            CpiContext::new_with_signer(
                ctx.accounts.system_program.to_account_info(),
                system_program::Transfer {
                    from: ctx.accounts.vault.to_account_info(),
                    to: ctx.accounts.admin.to_account_info(),
                },
                seeds,
            ),
            amount,
        )?;

        msg!("Admin withdrew {} lamports from vault", amount);
        Ok(())
    }

    /// Admin updates the SOL/USD price used for chip calculations.
    ///
    /// `new_sol_per_usd` uses 9-decimal precision (lamports per 1 USD).
    pub fn update_price(ctx: Context<UpdatePrice>, new_sol_per_usd: u64) -> Result<()> {
        require!(new_sol_per_usd > 0, ChipError::InvalidPrice);

        let game_state = &mut ctx.accounts.game_state;
        let old_price = game_state.sol_per_usd;
        game_state.sol_per_usd = new_sol_per_usd;

        msg!(
            "Price updated: {} -> {} lamports per USD",
            old_price,
            new_sol_per_usd
        );
        Ok(())
    }
}

// ---------------------------------------------------------------------------
// Account structs
// ---------------------------------------------------------------------------

#[derive(Accounts)]
pub struct Initialize<'info> {
    /// The admin who controls the game state and can withdraw funds.
    #[account(mut)]
    pub admin: Signer<'info>,

    /// Global game configuration, one per program.
    #[account(
        init,
        payer = admin,
        space = 8 + GameState::INIT_SPACE,
        seeds = [b"game_state"],
        bump,
    )]
    pub game_state: Account<'info, GameState>,

    /// PDA vault that holds all deposited SOL.
    /// We allocate zero data -- it only holds lamports.
    #[account(
        init,
        payer = admin,
        space = 0,
        seeds = [b"vault"],
        bump,
    )]
    /// CHECK: Vault PDA used solely to hold SOL. No data is stored.
    pub vault: AccountInfo<'info>,

    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct BuyChips<'info> {
    /// The player purchasing chips.
    #[account(mut)]
    pub buyer: Signer<'info>,

    /// Global game state (read for price, mutated for totals).
    #[account(
        mut,
        seeds = [b"game_state"],
        bump = game_state.state_bump,
    )]
    pub game_state: Account<'info, GameState>,

    /// PDA vault receiving SOL from the buyer.
    #[account(
        mut,
        seeds = [b"vault"],
        bump = game_state.vault_bump,
    )]
    /// CHECK: Vault PDA used solely to hold SOL.
    pub vault: AccountInfo<'info>,

    /// Per-user account tracking purchase history.
    /// Created on first purchase via `init_if_needed`.
    #[account(
        init_if_needed,
        payer = buyer,
        space = 8 + UserAccount::INIT_SPACE,
        seeds = [b"user_account", buyer.key().as_ref()],
        bump,
    )]
    pub user_account: Account<'info, UserAccount>,

    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct Withdraw<'info> {
    /// Only the admin stored in game_state may withdraw.
    #[account(
        mut,
        constraint = admin.key() == game_state.admin @ ChipError::Unauthorized,
    )]
    pub admin: Signer<'info>,

    #[account(
        seeds = [b"game_state"],
        bump = game_state.state_bump,
    )]
    pub game_state: Account<'info, GameState>,

    #[account(
        mut,
        seeds = [b"vault"],
        bump = game_state.vault_bump,
    )]
    /// CHECK: Vault PDA used solely to hold SOL.
    pub vault: AccountInfo<'info>,

    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct UpdatePrice<'info> {
    /// Only the admin may update the price.
    #[account(
        constraint = admin.key() == game_state.admin @ ChipError::Unauthorized,
    )]
    pub admin: Signer<'info>,

    #[account(
        mut,
        seeds = [b"game_state"],
        bump = game_state.state_bump,
    )]
    pub game_state: Account<'info, GameState>,
}

// ---------------------------------------------------------------------------
// State accounts
// ---------------------------------------------------------------------------

/// Global game configuration stored at a PDA.
#[account]
#[derive(InitSpace)]
pub struct GameState {
    /// Public key of the admin who can withdraw and update price.
    pub admin: Pubkey,
    /// How many lamports equal 1 USD (9-decimal precision).
    /// Example: if SOL = $200, then sol_per_usd = 5_000_000 (0.005 SOL).
    pub sol_per_usd: u64,
    /// Running total of lamports deposited into the vault.
    pub total_sol_collected: u64,
    /// Running total of game chips sold.
    pub total_chips_sold: u64,
    /// Bump seed for the vault PDA.
    pub vault_bump: u8,
    /// Bump seed for this game_state PDA.
    pub state_bump: u8,
}

/// Per-user purchase tracking stored at a PDA seeded by the user's pubkey.
#[account]
#[derive(InitSpace)]
pub struct UserAccount {
    /// Owner of this account (the player).
    pub owner: Pubkey,
    /// Cumulative chips purchased.
    pub total_chips_purchased: u64,
    /// Cumulative SOL spent (in lamports).
    pub total_sol_spent: u64,
    /// Slot of the most recent purchase.
    pub last_purchase_slot: u64,
    /// Bump seed for this PDA.
    pub bump: u8,
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

#[error_code]
pub enum ChipError {
    #[msg("Only the admin may perform this action.")]
    Unauthorized,
    #[msg("Amount must be greater than zero.")]
    ZeroAmount,
    #[msg("SOL/USD price must be greater than zero.")]
    InvalidPrice,
    #[msg("Arithmetic overflow during chip calculation.")]
    MathOverflow,
    #[msg("Purchase amount too small to yield any chips.")]
    PurchaseTooSmall,
    #[msg("Vault does not have enough SOL for this withdrawal.")]
    InsufficientVaultBalance,
}
